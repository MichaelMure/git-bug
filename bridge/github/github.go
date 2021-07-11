// Package github contains the Github bridge implementation
package github

import (
	"context"
	"time"

	"golang.org/x/oauth2"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
)

const (
	target = "github"

	metaKeyGithubId    = "github-id"
	metaKeyGithubUrl   = "github-url"
	metaKeyGithubLogin = "github-login"

	confKeyOwner        = "owner"
	confKeyProject      = "project"
	confKeyDefaultLogin = "default-login"

	githubV3Url    = "https://api.github.com"
	defaultTimeout = 60 * time.Second
)

var _ core.BridgeImpl = &Github{}

type Github struct{}

func (*Github) Target() string {
	return target
}

func (g *Github) LoginMetaKey() string {
	return metaKeyGithubLogin
}

func (*Github) NewImporter() core.Importer {
	return &githubImporter{}
}

func (*Github) NewExporter() core.Exporter {
	return &githubExporter{}
}

func buildClient(token *auth.Token) *client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token.Value},
	)
	httpClient := oauth2.NewClient(context.TODO(), src)

	return newClient(httpClient)
}
