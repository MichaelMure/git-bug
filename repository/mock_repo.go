package repository

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"strings"
	"sync"

	"github.com/99designs/keyring"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ ClockedRepo = &mockRepo{}
var _ TestedRepo = &mockRepo{}

// mockRepo defines an instance of Repo that can be used for testing.
type mockRepo struct {
	*mockRepoConfig
	*mockRepoKeyring
	*mockRepoCommon
	*mockRepoStorage
	*mockRepoIndex
	*mockRepoData
	*mockRepoClock
	*mockRepoTest
}

func (m *mockRepo) Close() error { return nil }

func NewMockRepo() *mockRepo {
	return &mockRepo{
		mockRepoConfig:  NewMockRepoConfig(),
		mockRepoKeyring: NewMockRepoKeyring(),
		mockRepoCommon:  NewMockRepoCommon(),
		mockRepoStorage: NewMockRepoStorage(),
		mockRepoIndex:   newMockRepoIndex(),
		mockRepoData:    NewMockRepoData(),
		mockRepoClock:   NewMockRepoClock(),
		mockRepoTest:    NewMockRepoTest(),
	}
}

var _ RepoConfig = &mockRepoConfig{}

type mockRepoConfig struct {
	localConfig  *MemConfig
	globalConfig *MemConfig
}

func NewMockRepoConfig() *mockRepoConfig {
	return &mockRepoConfig{
		localConfig:  NewMemConfig(),
		globalConfig: NewMemConfig(),
	}
}

// LocalConfig give access to the repository scoped configuration
func (r *mockRepoConfig) LocalConfig() Config {
	return r.localConfig
}

// GlobalConfig give access to the git global configuration
func (r *mockRepoConfig) GlobalConfig() Config {
	return r.globalConfig
}

// AnyConfig give access to a merged local/global configuration
func (r *mockRepoConfig) AnyConfig() ConfigRead {
	return mergeConfig(r.localConfig, r.globalConfig)
}

var _ RepoKeyring = &mockRepoKeyring{}

type mockRepoKeyring struct {
	keyring *keyring.ArrayKeyring
}

func NewMockRepoKeyring() *mockRepoKeyring {
	return &mockRepoKeyring{
		keyring: keyring.NewArrayKeyring(nil),
	}
}

// Keyring give access to a user-wide storage for secrets
func (r *mockRepoKeyring) Keyring() Keyring {
	return r.keyring
}

var _ RepoCommon = &mockRepoCommon{}

type mockRepoCommon struct{}

func NewMockRepoCommon() *mockRepoCommon {
	return &mockRepoCommon{}
}

func (r *mockRepoCommon) GetUserName() (string, error) {
	return "René Descartes", nil
}

// GetUserEmail returns the email address that the user has used to configure git.
func (r *mockRepoCommon) GetUserEmail() (string, error) {
	return "user@example.com", nil
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (r *mockRepoCommon) GetCoreEditor() (string, error) {
	return "vi", nil
}

// GetRemotes returns the configured remotes repositories.
func (r *mockRepoCommon) GetRemotes() (map[string]string, error) {
	return map[string]string{
		"origin": "git://github.com/MichaelMure/git-bug",
	}, nil
}

var _ RepoStorage = &mockRepoStorage{}

type mockRepoStorage struct {
	localFs LocalStorage
}

func NewMockRepoStorage() *mockRepoStorage {
	return &mockRepoStorage{localFs: billyLocalStorage{Filesystem: memfs.New()}}
}

func (m *mockRepoStorage) LocalStorage() LocalStorage {
	return m.localFs
}

var _ RepoIndex = &mockRepoIndex{}

type mockRepoIndex struct {
	indexesMutex sync.Mutex
	indexes      map[string]Index
}

func newMockRepoIndex() *mockRepoIndex {
	return &mockRepoIndex{
		indexes: make(map[string]Index),
	}
}

func (m *mockRepoIndex) GetIndex(name string) (Index, error) {
	m.indexesMutex.Lock()
	defer m.indexesMutex.Unlock()

	if index, ok := m.indexes[name]; ok {
		return index, nil
	}

	index := newIndex()
	m.indexes[name] = index
	return index, nil
}

var _ Index = &mockIndex{}

type mockIndex map[string][]string

func newIndex() *mockIndex {
	m := make(map[string][]string)
	return (*mockIndex)(&m)
}

func (m *mockIndex) IndexOne(id string, texts []string) error {
	(*m)[id] = texts
	return nil
}

func (m *mockIndex) IndexBatch() (indexer func(id string, texts []string) error, closer func() error) {
	indexer = func(id string, texts []string) error {
		(*m)[id] = texts
		return nil
	}
	closer = func() error { return nil }
	return indexer, closer
}

func (m *mockIndex) Search(terms []string) (ids []string, err error) {
loop:
	for id, texts := range *m {
		for _, text := range texts {
			for _, s := range strings.Fields(text) {
				for _, term := range terms {
					if s == term {
						ids = append(ids, id)
						continue loop
					}
				}
			}
		}
	}
	return ids, nil
}

func (m *mockIndex) DocCount() (uint64, error) {
	return uint64(len(*m)), nil
}

func (m *mockIndex) Remove(id string) error {
	delete(*m, id)
	return nil
}

func (m *mockIndex) Clear() error {
	for k, _ := range *m {
		delete(*m, k)
	}
	return nil
}

func (m *mockIndex) Close() error {
	return nil
}

var _ RepoData = &mockRepoData{}

type commit struct {
	treeHash Hash
	parents  []Hash
	sig      string
}

type mockRepoData struct {
	blobs   map[Hash][]byte
	trees   map[Hash]string
	commits map[Hash]commit
	refs    map[string]Hash
}

func NewMockRepoData() *mockRepoData {
	return &mockRepoData{
		blobs:   make(map[Hash][]byte),
		trees:   make(map[Hash]string),
		commits: make(map[Hash]commit),
		refs:    make(map[string]Hash),
	}
}

func (r *mockRepoData) FetchRefs(remote string, prefixes ...string) (string, error) {
	panic("implement me")
}

// PushRefs push git refs to a remote
func (r *mockRepoData) PushRefs(remote string, prefixes ...string) (string, error) {
	panic("implement me")
}

func (r *mockRepoData) SSHAuth(remote string) (*ssh.PublicKeys, error) {
	panic("implement me")
}

func (r *mockRepoData) StoreData(data []byte) (Hash, error) {
	rawHash := sha1.Sum(data)
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.blobs[hash] = data
	return hash, nil
}

func (r *mockRepoData) ReadData(hash Hash) ([]byte, error) {
	data, ok := r.blobs[hash]
	if !ok {
		return nil, ErrNotFound
	}

	return data, nil
}

func (r *mockRepoData) StoreTree(entries []TreeEntry) (Hash, error) {
	buffer := prepareTreeEntries(entries)
	rawHash := sha1.Sum(buffer.Bytes())
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.trees[hash] = buffer.String()

	return hash, nil
}

func (r *mockRepoData) ReadTree(hash Hash) ([]TreeEntry, error) {
	var data string

	data, ok := r.trees[hash]

	if !ok {
		// Git will understand a commit hash to reach a tree
		commit, ok := r.commits[hash]

		if !ok {
			return nil, ErrNotFound
		}

		data, ok = r.trees[commit.treeHash]

		if !ok {
			return nil, ErrNotFound
		}
	}

	return readTreeEntries(data)
}

func (r *mockRepoData) StoreCommit(treeHash Hash, parents ...Hash) (Hash, error) {
	return r.StoreSignedCommit(treeHash, nil, parents...)
}

func (r *mockRepoData) StoreSignedCommit(treeHash Hash, signKey *openpgp.Entity, parents ...Hash) (Hash, error) {
	hasher := sha1.New()
	hasher.Write([]byte(treeHash))
	for _, parent := range parents {
		hasher.Write([]byte(parent))
	}
	rawHash := hasher.Sum(nil)
	hash := Hash(fmt.Sprintf("%x", rawHash))
	c := commit{
		treeHash: treeHash,
		parents:  parents,
	}
	if signKey != nil {
		// unlike go-git, we only sign the tree hash for simplicity instead of all the fields (parents ...)
		var sig bytes.Buffer
		if err := openpgp.DetachSign(&sig, signKey, strings.NewReader(string(treeHash)), nil); err != nil {
			return "", err
		}
		c.sig = sig.String()
	}
	r.commits[hash] = c
	return hash, nil
}

func (r *mockRepoData) ReadCommit(hash Hash) (Commit, error) {
	c, ok := r.commits[hash]
	if !ok {
		return Commit{}, ErrNotFound
	}

	result := Commit{
		Hash:     hash,
		Parents:  c.parents,
		TreeHash: c.treeHash,
	}

	if c.sig != "" {
		// Note: this is actually incorrect as the signed data should be the full commit (+comment, +date ...)
		// but only the tree hash work for our purpose here.
		result.SignedData = strings.NewReader(string(c.treeHash))
		result.Signature = strings.NewReader(c.sig)
	}

	return result, nil
}

func (r *mockRepoData) ResolveRef(ref string) (Hash, error) {
	h, ok := r.refs[ref]
	if !ok {
		return "", ErrNotFound
	}
	return h, nil
}

func (r *mockRepoData) UpdateRef(ref string, hash Hash) error {
	r.refs[ref] = hash
	return nil
}

func (r *mockRepoData) RemoveRef(ref string) error {
	delete(r.refs, ref)
	return nil
}

func (r *mockRepoData) ListRefs(refPrefix string) ([]string, error) {
	var keys []string

	for k := range r.refs {
		if strings.HasPrefix(k, refPrefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (r *mockRepoData) RefExist(ref string) (bool, error) {
	_, exist := r.refs[ref]
	return exist, nil
}

func (r *mockRepoData) CopyRef(source string, dest string) error {
	hash, exist := r.refs[source]

	if !exist {
		return ErrNotFound
	}

	r.refs[dest] = hash
	return nil
}

func (r *mockRepoData) ListCommits(ref string) ([]Hash, error) {
	return nonNativeListCommits(r, ref)
}

var _ RepoClock = &mockRepoClock{}

type mockRepoClock struct {
	mu     sync.Mutex
	clocks map[string]lamport.Clock
}

func NewMockRepoClock() *mockRepoClock {
	return &mockRepoClock{
		clocks: make(map[string]lamport.Clock),
	}
}

func (r *mockRepoClock) AllClocks() (map[string]lamport.Clock, error) {
	return r.clocks, nil
}

func (r *mockRepoClock) GetOrCreateClock(name string) (lamport.Clock, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.clocks[name]; ok {
		return c, nil
	}

	c := lamport.NewMemClock()
	r.clocks[name] = c
	return c, nil
}

func (r *mockRepoClock) Increment(name string) (lamport.Time, error) {
	c, err := r.GetOrCreateClock(name)
	if err != nil {
		return lamport.Time(0), err
	}
	return c.Increment()
}

func (r *mockRepoClock) Witness(name string, time lamport.Time) error {
	c, err := r.GetOrCreateClock(name)
	if err != nil {
		return err
	}
	return c.Witness(time)
}

var _ repoTest = &mockRepoTest{}

type mockRepoTest struct{}

func NewMockRepoTest() *mockRepoTest {
	return &mockRepoTest{}
}

func (r *mockRepoTest) AddRemote(name string, url string) error {
	panic("implement me")
}

func (r mockRepoTest) GetLocalRemote() string {
	panic("implement me")
}

func (r mockRepoTest) EraseFromDisk() error {
	// nothing to do
	return nil
}
