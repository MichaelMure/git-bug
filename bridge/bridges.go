// Package bridge contains the high-level public functions to use and manage bridges
package bridge

import (
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/github"
	"github.com/MichaelMure/git-bug/bridge/gitlab"
	"github.com/MichaelMure/git-bug/bridge/jira"
	"github.com/MichaelMure/git-bug/bridge/launchpad"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

func init() {
	core.Register(&github.Github{})
	core.Register(&gitlab.Gitlab{})
	core.Register(&launchpad.Launchpad{})
	core.Register(&jira.Jira{})
}

// Targets return all known bridge implementation target
func Targets() []string {
	return core.Targets()
}

// ValidParams returns the valid input parameters for configuring a target bridge
func ValidParams(target string) ([]string, error) {
	return core.ValidParams(target)
}

// LoginMetaKey return the metadata key used to store the remote bug-tracker login
// on the user identity. The corresponding value is used to match identities and
// credentials.
func LoginMetaKey(target string) (string, error) {
	return core.LoginMetaKey(target)
}

// Instantiate a new Bridge for a repo, from the given target and name
func NewBridge(repo *cache.RepoCache, target string, name string) (*core.Bridge, error) {
	return core.NewBridge(repo, target, name)
}

// LoadBridge instantiate a new bridge from a repo configuration
func LoadBridge(repo *cache.RepoCache, name string) (*core.Bridge, error) {
	return core.LoadBridge(repo, name)
}

// Attempt to retrieve a default bridge for the given repo. If zero or multiple
// bridge exist, it fails.
func DefaultBridge(repo *cache.RepoCache) (*core.Bridge, error) {
	return core.DefaultBridge(repo)
}

// ConfiguredBridges return the list of bridge that are configured for the given
// repo
func ConfiguredBridges(repo repository.RepoConfig) ([]string, error) {
	return core.ConfiguredBridges(repo)
}

// Remove a configured bridge
func RemoveBridge(repo repository.RepoConfig, name string) error {
	return core.RemoveBridge(repo, name)
}
