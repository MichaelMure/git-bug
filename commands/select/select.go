package _select

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
)

type ErrNoValidId struct {
	typename string
}

func NewErrNoValidId(typename string) *ErrNoValidId {
	return &ErrNoValidId{typename: typename}
}

func (e ErrNoValidId) Error() string {
	return fmt.Sprintf("you must provide a %s id or use the \"select\" command first", e.typename)
}

func IsErrNoValidId(err error) bool {
	_, ok := err.(*ErrNoValidId)
	return ok
}

type Resolver[CacheT cache.CacheEntity] interface {
	Resolve(id entity.Id) (CacheT, error)
	ResolvePrefix(prefix string) (CacheT, error)
}

// Resolve first try to resolve an entity using the first argument of the command
// line. If it fails, it falls back to the select mechanism.
//
// Returns:
//
// Contrary to golang convention, the list of args returned is still correct even in
// case of error, which allows to keep going and decide to handle the failure case more
// naturally.
func Resolve[CacheT cache.CacheEntity](repo *cache.RepoCache,
	typename string, namespace string, resolver Resolver[CacheT],
	args []string) (CacheT, []string, error) {
	// At first, try to use the first argument as an entity prefix
	if len(args) > 0 {
		cached, err := resolver.ResolvePrefix(args[0])

		if err == nil {
			return cached, args[1:], nil
		}

		if !entity.IsErrNotFound(err) {
			return *new(CacheT), args, err
		}
	}

	// first arg is not a valid entity prefix, we can safely use the preselected entity if any

	cached, err := selected(repo, resolver, namespace)

	// selected entity is invalid
	if entity.IsErrNotFound(err) {
		// we clear the selected bug
		err = Clear(repo, namespace)
		if err != nil {
			return *new(CacheT), args, err
		}
		return *new(CacheT), args, NewErrNoValidId(typename)
	}

	// another error when reading the entity
	if err != nil {
		return *new(CacheT), args, err
	}

	// entity is successfully retrieved
	if cached != nil {
		return *cached, args, nil
	}

	// no selected bug and no valid first argument
	return *new(CacheT), args, NewErrNoValidId(typename)
}

func selectFileName(namespace string) string {
	return filepath.Join("select", namespace)
}

// Select will select a bug for future use
func Select(repo *cache.RepoCache, namespace string, id entity.Id) error {
	filename := selectFileName(namespace)
	f, err := repo.LocalStorage().OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(id.String()))
	if err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

// Clear will clear the selected entity, if any
func Clear(repo *cache.RepoCache, namespace string) error {
	filename := selectFileName(namespace)
	return repo.LocalStorage().Remove(filename)
}

func selected[CacheT cache.CacheEntity](repo *cache.RepoCache, resolver Resolver[CacheT], namespace string) (*CacheT, error) {
	filename := selectFileName(namespace)
	f, err := repo.LocalStorage().Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	buf, err := io.ReadAll(io.LimitReader(f, 100))
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	if len(buf) >= 100 {
		return nil, fmt.Errorf("the select file should be < 100 bytes")
	}

	id := entity.Id(buf)
	if err := id.Validate(); err != nil {
		err = repo.LocalStorage().Remove(filename)
		if err != nil {
			return nil, errors.Wrap(err, "error while removing invalid select file")
		}

		return nil, fmt.Errorf("select file in invalid, removing it")
	}

	cached, err := resolver.Resolve(id)
	if err != nil {
		return nil, err
	}

	return &cached, nil
}
