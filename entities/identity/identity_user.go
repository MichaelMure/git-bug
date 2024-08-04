package identity

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	bootstrap "github.com/MichaelMure/git-bug/entity/boostrap"
	"github.com/MichaelMure/git-bug/repository"
)

// SetUserIdentity store the user identity's id in the git config
func SetUserIdentity(repo repository.RepoConfig, identity *Identity) error {
	return repo.LocalConfig().StoreString(identityConfigKey, identity.Id().String())
}

func ClearUserIdentity(repo repository.RepoConfig) error {
	return repo.LocalConfig().RemoveAll(identityConfigKey)
}

// GetUserIdentity read the current user identity, set with a git config entry
func GetUserIdentity(repo repository.Repo) (*Identity, error) {
	id, err := GetUserIdentityId(repo)
	if err != nil {
		return nil, err
	}

	i, err := ReadLocal(repo, id)
	if bootstrap.IsErrNotFound(err) {
		innerErr := repo.LocalConfig().RemoveAll(identityConfigKey)
		if innerErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, errors.Wrap(innerErr, "can't clear user identity").Error())
		}
		return nil, err
	}

	return i, nil
}

func GetUserIdentityId(repo repository.Repo) (bootstrap.Id, error) {
	val, err := repo.LocalConfig().ReadString(identityConfigKey)
	if errors.Is(err, repository.ErrNoConfigEntry) {
		return bootstrap.UnsetId, ErrNoIdentitySet
	}
	if errors.Is(err, repository.ErrMultipleConfigEntry) {
		return bootstrap.UnsetId, ErrMultipleIdentitiesSet
	}
	if err != nil {
		return bootstrap.UnsetId, err
	}

	var id = bootstrap.Id(val)

	if err := id.Validate(); err != nil {
		return bootstrap.UnsetId, err
	}

	return id, nil
}

// IsUserIdentitySet say if the user has set his identity
func IsUserIdentitySet(repo repository.Repo) (bool, error) {
	_, err := repo.LocalConfig().ReadString(identityConfigKey)
	if errors.Is(err, repository.ErrNoConfigEntry) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
