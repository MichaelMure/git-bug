package repository

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-billy/v5/osfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/execabs"

	"github.com/MichaelMure/git-bug/util/lamport"
)

const clockPath = "clocks"
const indexPath = "indexes"

var _ ClockedRepo = &GoGitRepo{}
var _ TestedRepo = &GoGitRepo{}

type GoGitRepo struct {
	// Unfortunately, some parts of go-git are not thread-safe so we have to cover them with a big fat mutex here.
	// See https://github.com/go-git/go-git/issues/48
	// See https://github.com/go-git/go-git/issues/208
	// See https://github.com/go-git/go-git/pull/186
	rMutex sync.Mutex
	r      *gogit.Repository
	path   string

	clocksMutex sync.Mutex
	clocks      map[string]lamport.Clock

	indexesMutex sync.Mutex
	indexes      map[string]Index

	keyring      Keyring
	localStorage LocalStorage
}

// OpenGoGitRepo opens an already existing repo at the given path and
// with the specified LocalStorage namespace.  Given a repository path
// of "~/myrepo" and a namespace of "git-bug", local storage for the
// GoGitRepo will be configured at "~/myrepo/.git/git-bug".
func OpenGoGitRepo(path, namespace string, clockLoaders []ClockLoader) (*GoGitRepo, error) {
	path, err := detectGitPath(path, 0)
	if err != nil {
		return nil, err
	}

	r, err := gogit.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	repo := &GoGitRepo{
		r:            r,
		path:         path,
		clocks:       make(map[string]lamport.Clock),
		indexes:      make(map[string]Index),
		keyring:      k,
		localStorage: billyLocalStorage{Filesystem: osfs.New(filepath.Join(path, namespace))},
	}

	loaderToRun := make([]ClockLoader, 0, len(clockLoaders))
	for _, loader := range clockLoaders {
		loader := loader
		allExist := true
		for _, name := range loader.Clocks {
			if _, err := repo.getClock(name); err != nil {
				allExist = false
			}
		}

		if !allExist {
			loaderToRun = append(loaderToRun, loader)
		}
	}

	var errG errgroup.Group
	for _, loader := range loaderToRun {
		loader := loader
		errG.Go(func() error {
			return loader.Witnesser(repo)
		})
	}
	err = errG.Wait()
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// InitGoGitRepo creates a new empty git repo at the given path and
// with the specified LocalStorage namespace.  Given a repository path
// of "~/myrepo" and a namespace of "git-bug", local storage for the
// GoGitRepo will be configured at "~/myrepo/.git/git-bug".
func InitGoGitRepo(path, namespace string) (*GoGitRepo, error) {
	r, err := gogit.PlainInit(path, false)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:            r,
		path:         filepath.Join(path, ".git"),
		clocks:       make(map[string]lamport.Clock),
		indexes:      make(map[string]Index),
		keyring:      k,
		localStorage: billyLocalStorage{Filesystem: osfs.New(filepath.Join(path, ".git", namespace))},
	}, nil
}

// InitBareGoGitRepo creates a new --bare empty git repo at the given
// path and with the specified LocalStorage namespace.  Given a repository
// path of "~/myrepo" and a namespace of "git-bug", local storage for the
// GoGitRepo will be configured at "~/myrepo/.git/git-bug".
func InitBareGoGitRepo(path, namespace string) (*GoGitRepo, error) {
	r, err := gogit.PlainInit(path, true)
	if err != nil {
		return nil, err
	}

	k, err := defaultKeyring()
	if err != nil {
		return nil, err
	}

	return &GoGitRepo{
		r:            r,
		path:         path,
		clocks:       make(map[string]lamport.Clock),
		indexes:      make(map[string]Index),
		keyring:      k,
		localStorage: billyLocalStorage{Filesystem: osfs.New(filepath.Join(path, namespace))},
	}, nil
}

func detectGitPath(path string, depth int) (string, error) {
	if depth >= 10 {
		return "", fmt.Errorf("gitdir loop detected")
	}

	// normalize the path
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		fi, err := os.Stat(filepath.Join(path, ".git"))
		if err == nil {
			if !fi.IsDir() {
				// See if our .git item is a dotfile that holds a submodule reference
				dotfile, err := os.Open(filepath.Join(path, fi.Name()))
				if err != nil {
					// Can't open error
					return "", fmt.Errorf(".git exists but is not a directory or a readable file: %w", err)
				}
				// We aren't going to defer the dotfile.Close, because we might keep looping, so we have to be sure to
				// clean up before returning an error
				reader := bufio.NewReader(io.LimitReader(dotfile, 2048))
				line, _, err := reader.ReadLine()
				_ = dotfile.Close()
				if err != nil {
					return "", fmt.Errorf(".git exists but is not a directory and cannot be read: %w", err)
				}
				dotContent := string(line)
				if strings.HasPrefix(dotContent, "gitdir:") {
					// This is a submodule parent path link. Strip the prefix, clean the string of whitespace just to
					// be safe, and return
					dotContent = strings.TrimSpace(strings.TrimPrefix(dotContent, "gitdir: "))
					p, err := detectGitPath(dotContent, depth+1)
					if err != nil {
						return "", fmt.Errorf(".git gitdir error: %w", err)
					}
					return p, nil
				}
				return "", fmt.Errorf(".git exist but is not a directory or module/workspace file")
			}
			return filepath.Join(path, ".git"), nil
		}
		if !os.IsNotExist(err) {
			// unknown error
			return "", err
		}

		// detect bare repo
		ok, err := isGitDir(path)
		if err != nil {
			return "", err
		}
		if ok {
			return path, nil
		}

		if parent := filepath.Dir(path); parent == path {
			return "", fmt.Errorf(".git not found")
		} else {
			path = parent
		}
	}
}

func isGitDir(path string) (bool, error) {
	markers := []string{"HEAD", "objects", "refs"}

	for _, marker := range markers {
		_, err := os.Stat(filepath.Join(path, marker))
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			// unknown error
			return false, err
		} else {
			return false, nil
		}
	}

	return true, nil
}

func (repo *GoGitRepo) Close() error {
	var firstErr error
	for name, index := range repo.indexes {
		err := index.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		delete(repo.indexes, name)
	}
	return firstErr
}

// LocalConfig give access to the repository scoped configuration
func (repo *GoGitRepo) LocalConfig() Config {
	return newGoGitLocalConfig(repo.r)
}

// GlobalConfig give access to the global scoped configuration
func (repo *GoGitRepo) GlobalConfig() Config {
	return newGoGitGlobalConfig()
}

// AnyConfig give access to a merged local/global configuration
func (repo *GoGitRepo) AnyConfig() ConfigRead {
	return mergeConfig(repo.LocalConfig(), repo.GlobalConfig())
}

// Keyring give access to a user-wide storage for secrets
func (repo *GoGitRepo) Keyring() Keyring {
	return repo.keyring
}

// GetUserName returns the name the user has used to configure git
func (repo *GoGitRepo) GetUserName() (string, error) {
	return repo.AnyConfig().ReadString("user.name")
}

// GetUserEmail returns the email address that the user has used to configure git.
func (repo *GoGitRepo) GetUserEmail() (string, error) {
	return repo.AnyConfig().ReadString("user.email")
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (repo *GoGitRepo) GetCoreEditor() (string, error) {
	// See https://git-scm.com/docs/git-var
	// The order of preference is the $GIT_EDITOR environment variable, then core.editor configuration, then $VISUAL, then $EDITOR, and then the default chosen at compile time, which is usually vi.

	if val, ok := os.LookupEnv("GIT_EDITOR"); ok {
		return val, nil
	}

	val, err := repo.AnyConfig().ReadString("core.editor")
	if err == nil && val != "" {
		return val, nil
	}
	if err != nil && !errors.Is(err, ErrNoConfigEntry) {
		return "", err
	}

	if val, ok := os.LookupEnv("VISUAL"); ok {
		return val, nil
	}

	if val, ok := os.LookupEnv("EDITOR"); ok {
		return val, nil
	}

	priorities := []string{
		"editor",
		"nano",
		"vim",
		"vi",
		"emacs",
	}

	for _, cmd := range priorities {
		if _, err = execabs.LookPath(cmd); err == nil {
			return cmd, nil
		}

	}

	return "ed", nil
}

// GetRemotes returns the configured remotes repositories.
func (repo *GoGitRepo) GetRemotes() (map[string]string, error) {
	cfg, err := repo.r.Config()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(cfg.Remotes))
	for name, remote := range cfg.Remotes {
		if len(remote.URLs) > 0 {
			result[name] = remote.URLs[0]
		}
	}

	return result, nil
}

// LocalStorage returns a billy.Filesystem giving access to
// $RepoPath/.git/$Namespace.
func (repo *GoGitRepo) LocalStorage() LocalStorage {
	return repo.localStorage
}

func (repo *GoGitRepo) GetIndex(name string) (Index, error) {
	repo.indexesMutex.Lock()
	defer repo.indexesMutex.Unlock()

	if index, ok := repo.indexes[name]; ok {
		return index, nil
	}

	path := filepath.Join(repo.localStorage.Root(), indexPath, name)

	index, err := openBleveIndex(path)
	if err == nil {
		repo.indexes[name] = index
	}
	return index, err
}

// FetchRefs fetch git refs matching a directory prefix to a remote
// Ex: prefix="foo" will fetch any remote refs matching "refs/foo/*" locally.
// The equivalent git refspec would be "refs/foo/*:refs/remotes/<remote>/foo/*"
func (repo *GoGitRepo) FetchRefs(remote string, prefixes ...string) (string, error) {
	refSpecs := make([]config.RefSpec, len(prefixes))

	for i, prefix := range prefixes {
		refSpecs[i] = config.RefSpec(fmt.Sprintf("refs/%s/*:refs/remotes/%s/%s/*", prefix, remote, prefix))
	}

	buf := bytes.NewBuffer(nil)

	fetchOptions := &gogit.FetchOptions{
		RemoteName: remote,
		RefSpecs:   refSpecs,
		Progress:   buf,
	}

	publicKeys, err := repo.SSHAuth(remote)
	if err != nil {
		return "", err
	}

	err = repo.r.Fetch(fetchOptions)
	if err == gogit.NoErrAlreadyUpToDate {
		return "already up-to-date", nil
	}
	// ssh-agent is required if ssh or scp-like url is configured in repository config
	// we can not fetch if the ssh-agent has invalid keys or ssh-agent is not running
	// retry to fetch again if we can retreive public keys from the users home directory
	if err != nil && publicKeys != nil {
		fetchOptions.Auth = publicKeys
		err = repo.r.Fetch(fetchOptions)
		if err == gogit.NoErrAlreadyUpToDate {
			return "already up-to-date", nil
		}
	}
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// PushRefs push git refs matching a directory prefix to a remote
// Ex: prefix="foo" will push any local refs matching "refs/foo/*" to the remote.
// The equivalent git refspec would be "refs/foo/*:refs/foo/*"
//
// Additionally, PushRefs will update the local references in refs/remotes/<remote>/foo to match
// the remote state.
func (repo *GoGitRepo) PushRefs(remote string, prefixes ...string) (string, error) {
	remo, err := repo.r.Remote(remote)
	if err != nil {
		return "", err
	}

	refSpecs := make([]config.RefSpec, len(prefixes))

	for i, prefix := range prefixes {
		refspec := fmt.Sprintf("refs/%s/*:refs/%s/*", prefix, prefix)

		// to make sure that the push also create the corresponding refs/remotes/<remote>/... references,
		// we need to have a default fetch refspec configured on the remote, to make our refs "track" the remote ones.
		// This does not change the config on disk, only on memory.
		hasCustomFetch := false
		fetchRefspec := fmt.Sprintf("refs/%s/*:refs/remotes/%s/%s/*", prefix, remote, prefix)
		for _, r := range remo.Config().Fetch {
			if string(r) == fetchRefspec {
				hasCustomFetch = true
				break
			}
		}

		if !hasCustomFetch {
			remo.Config().Fetch = append(remo.Config().Fetch, config.RefSpec(fetchRefspec))
		}

		refSpecs[i] = config.RefSpec(refspec)
	}

	buf := bytes.NewBuffer(nil)

	pushOptions := &gogit.PushOptions{
		RemoteName: remote,
		RefSpecs:   refSpecs,
		Progress:   buf,
	}

	publicKeys, err := repo.SSHAuth(remote)
	if err != nil {
		return "", err
	}

	err = repo.r.Push(pushOptions)
	if err == gogit.NoErrAlreadyUpToDate {
		return "already up-to-date", nil
	}
	// ssh-agent is required if ssh or scp-like url is configured in repository config
	// we can not push if the ssh-agent has invalid keys or ssh-agent is not running
	// retry to push again if we can retreive public keys from the users home directory
	if err != nil && publicKeys != nil {
		pushOptions.Auth = publicKeys
		err = repo.r.Push(pushOptions)
		if err == gogit.NoErrAlreadyUpToDate {
			return "already up-to-date", nil
		}
	}
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// SSHAuth will attempt to read public keys for SSH auth
// if the repository remote contains a ssh or scp-like url
func (repo *GoGitRepo) SSHAuth(remote string) (*ssh.PublicKeys, error) {
	// get the repository config
	config, err := repo.r.Config()
	if err != nil {
		return nil, err
	}

	// check if the repository config has at least one remote url
	remotes, found := config.Remotes[remote]
	if !found || len(remotes.URLs) < 1 {
		return nil, fmt.Errorf("remote %s url not found in repository config", remote)
	}

	schemeRegexp := regexp.MustCompile(`^[^:]+://`)
	scpLikeRegexp := regexp.MustCompile(`^(?:(?P<user>[^@]+)@)?(?P<host>)`)

	// try to load public keys from the users home directory
	// if the repository remote contains a ssh or scp-like url
	if strings.HasPrefix(remotes.URLs[0], "ssh://") || (scpLikeRegexp.MatchString(remotes.URLs[0]) && !schemeRegexp.MatchString(remotes.URLs[0])) {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		// try to find and load valid public keys from the users home directory
		attemptKeys := []string{"id_rsa", "id_ecdsa", "id_ecdsa_sk", "id_ed25519", "id_ed25519_sk", "id_xmss", "id_dsa"}
		for _, key := range attemptKeys {
			authMethod, err := ssh.NewPublicKeysFromFile("git", filepath.Join(home, ".ssh", key), "")
			if err == nil {
				return authMethod, nil
			}
		}
	}
	return nil, nil
}

// StoreData will store arbitrary data and return the corresponding hash
func (repo *GoGitRepo) StoreData(data []byte) (Hash, error) {
	obj := repo.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)

	w, err := obj.Writer()
	if err != nil {
		return "", err
	}

	_, err = w.Write(data)
	if err != nil {
		return "", err
	}

	h, err := repo.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	return Hash(h.String()), nil
}

// ReadData will attempt to read arbitrary data from the given hash
func (repo *GoGitRepo) ReadData(hash Hash) ([]byte, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	obj, err := repo.r.BlobObject(plumbing.NewHash(hash.String()))
	if err == plumbing.ErrObjectNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	r, err := obj.Reader()
	if err != nil {
		return nil, err
	}

	// TODO: return a io.Reader instead
	return io.ReadAll(r)
}

// StoreTree will store a mapping key-->Hash as a Git tree
func (repo *GoGitRepo) StoreTree(mapping []TreeEntry) (Hash, error) {
	var tree object.Tree

	// TODO: can be removed once https://github.com/go-git/go-git/issues/193 is resolved
	sorted := make([]TreeEntry, len(mapping))
	copy(sorted, mapping)
	sort.Slice(sorted, func(i, j int) bool {
		nameI := sorted[i].Name
		if sorted[i].ObjectType == Tree {
			nameI += "/"
		}
		nameJ := sorted[j].Name
		if sorted[j].ObjectType == Tree {
			nameJ += "/"
		}
		return nameI < nameJ
	})

	for _, entry := range sorted {
		mode := filemode.Regular
		if entry.ObjectType == Tree {
			mode = filemode.Dir
		}

		tree.Entries = append(tree.Entries, object.TreeEntry{
			Name: entry.Name,
			Mode: mode,
			Hash: plumbing.NewHash(entry.Hash.String()),
		})
	}

	obj := repo.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.TreeObject)
	err := tree.Encode(obj)
	if err != nil {
		return "", err
	}

	hash, err := repo.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	return Hash(hash.String()), nil
}

// ReadTree will return the list of entries in a Git tree
func (repo *GoGitRepo) ReadTree(hash Hash) ([]TreeEntry, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	h := plumbing.NewHash(hash.String())

	// the given hash could be a tree or a commit
	obj, err := repo.r.Storer.EncodedObject(plumbing.AnyObject, h)
	if err == plumbing.ErrObjectNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var tree *object.Tree
	switch obj.Type() {
	case plumbing.TreeObject:
		tree, err = object.DecodeTree(repo.r.Storer, obj)
	case plumbing.CommitObject:
		var commit *object.Commit
		commit, err = object.DecodeCommit(repo.r.Storer, obj)
		if err != nil {
			return nil, err
		}
		tree, err = commit.Tree()
	default:
		return nil, fmt.Errorf("given hash is not a tree")
	}
	if err != nil {
		return nil, err
	}

	treeEntries := make([]TreeEntry, len(tree.Entries))
	for i, entry := range tree.Entries {
		objType := Blob
		if entry.Mode == filemode.Dir {
			objType = Tree
		}

		treeEntries[i] = TreeEntry{
			ObjectType: objType,
			Hash:       Hash(entry.Hash.String()),
			Name:       entry.Name,
		}
	}

	return treeEntries, nil
}

// StoreCommit will store a Git commit with the given Git tree
func (repo *GoGitRepo) StoreCommit(treeHash Hash, parents ...Hash) (Hash, error) {
	return repo.StoreSignedCommit(treeHash, nil, parents...)
}

// StoreSignedCommit will store a Git commit with the given Git tree. If signKey is not nil, the commit
// will be signed accordingly.
func (repo *GoGitRepo) StoreSignedCommit(treeHash Hash, signKey *openpgp.Entity, parents ...Hash) (Hash, error) {
	cfg, err := repo.r.Config()
	if err != nil {
		return "", err
	}

	commit := object.Commit{
		Author: object.Signature{
			Name:  cfg.Author.Name,
			Email: cfg.Author.Email,
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  cfg.Committer.Name,
			Email: cfg.Committer.Email,
			When:  time.Now(),
		},
		Message:  "",
		TreeHash: plumbing.NewHash(treeHash.String()),
	}

	for _, parent := range parents {
		commit.ParentHashes = append(commit.ParentHashes, plumbing.NewHash(parent.String()))
	}

	// Compute the signature if needed
	if signKey != nil {
		// first get the serialized commit
		encoded := &plumbing.MemoryObject{}
		if err := commit.Encode(encoded); err != nil {
			return "", err
		}
		r, err := encoded.Reader()
		if err != nil {
			return "", err
		}

		// sign the data
		var sig bytes.Buffer
		if err := openpgp.ArmoredDetachSign(&sig, signKey, r, nil); err != nil {
			return "", err
		}
		commit.PGPSignature = sig.String()
	}

	obj := repo.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.CommitObject)
	err = commit.Encode(obj)
	if err != nil {
		return "", err
	}

	hash, err := repo.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	return Hash(hash.String()), nil
}

func (repo *GoGitRepo) ResolveRef(ref string) (Hash, error) {
	r, err := repo.r.Reference(plumbing.ReferenceName(ref), false)
	if err == plumbing.ErrReferenceNotFound {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return Hash(r.Hash().String()), nil
}

// UpdateRef will create or update a Git reference
func (repo *GoGitRepo) UpdateRef(ref string, hash Hash) error {
	return repo.r.Storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName(ref), plumbing.NewHash(hash.String())))
}

// RemoveRef will remove a Git reference
func (repo *GoGitRepo) RemoveRef(ref string) error {
	return repo.r.Storer.RemoveReference(plumbing.ReferenceName(ref))
}

// ListRefs will return a list of Git ref matching the given refspec
func (repo *GoGitRepo) ListRefs(refPrefix string) ([]string, error) {
	refIter, err := repo.r.References()
	if err != nil {
		return nil, err
	}

	refs := make([]string, 0)

	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if strings.HasPrefix(ref.Name().String(), refPrefix) {
			refs = append(refs, ref.Name().String())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return refs, nil
}

// RefExist will check if a reference exist in Git
func (repo *GoGitRepo) RefExist(ref string) (bool, error) {
	_, err := repo.r.Reference(plumbing.ReferenceName(ref), false)
	if err == nil {
		return true, nil
	} else if err == plumbing.ErrReferenceNotFound {
		return false, nil
	}
	return false, err
}

// CopyRef will create a new reference with the same value as another one
func (repo *GoGitRepo) CopyRef(source string, dest string) error {
	r, err := repo.r.Reference(plumbing.ReferenceName(source), false)
	if err == plumbing.ErrReferenceNotFound {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return repo.r.Storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName(dest), r.Hash()))
}

// ListCommits will return the list of tree hashes of a ref, in chronological order
func (repo *GoGitRepo) ListCommits(ref string) ([]Hash, error) {
	return nonNativeListCommits(repo, ref)
}

func (repo *GoGitRepo) ReadCommit(hash Hash) (Commit, error) {
	repo.rMutex.Lock()
	defer repo.rMutex.Unlock()

	commit, err := repo.r.CommitObject(plumbing.NewHash(hash.String()))
	if err == plumbing.ErrObjectNotFound {
		return Commit{}, ErrNotFound
	}
	if err != nil {
		return Commit{}, err
	}

	parents := make([]Hash, len(commit.ParentHashes))
	for i, parentHash := range commit.ParentHashes {
		parents[i] = Hash(parentHash.String())
	}

	result := Commit{
		Hash:     hash,
		Parents:  parents,
		TreeHash: Hash(commit.TreeHash.String()),
	}

	if commit.PGPSignature != "" {
		// I can't find a way to just remove the signature when reading the encoded commit so we need to
		// re-encode the commit without signature.

		encoded := &plumbing.MemoryObject{}
		err := commit.EncodeWithoutSignature(encoded)
		if err != nil {
			return Commit{}, err
		}

		result.SignedData, err = encoded.Reader()
		if err != nil {
			return Commit{}, err
		}

		result.Signature, err = deArmorSignature(strings.NewReader(commit.PGPSignature))
		if err != nil {
			return Commit{}, err
		}
	}

	return result, nil
}

func (repo *GoGitRepo) AllClocks() (map[string]lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	result := make(map[string]lamport.Clock)

	files, err := os.ReadDir(filepath.Join(repo.localStorage.Root(), clockPath))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		name := file.Name()
		if c, ok := repo.clocks[name]; ok {
			result[name] = c
		} else {
			c, err := lamport.LoadPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
			if err != nil {
				return nil, err
			}
			repo.clocks[name] = c
			result[name] = c
		}
	}

	return result, nil
}

// GetOrCreateClock return a Lamport clock stored in the Repo.
// If the clock doesn't exist, it's created.
func (repo *GoGitRepo) GetOrCreateClock(name string) (lamport.Clock, error) {
	repo.clocksMutex.Lock()
	defer repo.clocksMutex.Unlock()

	c, err := repo.getClock(name)
	if err == nil {
		return c, nil
	}
	if err != ErrClockNotExist {
		return nil, err
	}

	c, err = lamport.NewPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
	if err != nil {
		return nil, err
	}

	repo.clocks[name] = c
	return c, nil
}

func (repo *GoGitRepo) getClock(name string) (lamport.Clock, error) {
	if c, ok := repo.clocks[name]; ok {
		return c, nil
	}

	c, err := lamport.LoadPersistedClock(repo.LocalStorage(), filepath.Join(clockPath, name))
	if err == nil {
		repo.clocks[name] = c
		return c, nil
	}
	if err == lamport.ErrClockNotExist {
		return nil, ErrClockNotExist
	}
	return nil, err
}

// Increment is equivalent to c = GetOrCreateClock(name) + c.Increment()
func (repo *GoGitRepo) Increment(name string) (lamport.Time, error) {
	c, err := repo.GetOrCreateClock(name)
	if err != nil {
		return lamport.Time(0), err
	}
	return c.Increment()
}

// Witness is equivalent to c = GetOrCreateClock(name) + c.Witness(time)
func (repo *GoGitRepo) Witness(name string, time lamport.Time) error {
	c, err := repo.GetOrCreateClock(name)
	if err != nil {
		return err
	}
	return c.Witness(time)
}

// AddRemote add a new remote to the repository
// Not in the interface because it's only used for testing
func (repo *GoGitRepo) AddRemote(name string, url string) error {
	_, err := repo.r.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})

	return err
}

// GetLocalRemote return the URL to use to add this repo as a local remote
func (repo *GoGitRepo) GetLocalRemote() string {
	return repo.path
}

// EraseFromDisk delete this repository entirely from the disk
func (repo *GoGitRepo) EraseFromDisk() error {
	err := repo.Close()
	if err != nil {
		return err
	}

	path := filepath.Clean(strings.TrimSuffix(repo.path, string(filepath.Separator)+".git"))

	// fmt.Println("Cleaning repo:", path)
	return os.RemoveAll(path)
}
