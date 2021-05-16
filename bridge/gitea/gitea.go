package gitea

import (
	"time"

	"code.gitea.io/sdk/gitea"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
)

const (
	target = "gitea-preview"

	metaKeyGiteaID      = "gitea-id"
	metaKeyGiteaLogin   = "gitea-login"
	metaKeyGiteaOwner   = "gitea-owner"
	metaKeyGiteaProject = "gitea-project"
	metaKeyGiteaBaseURL = "gitea-base-url"

	confKeyOwner        = "owner"
	confKeyProject      = "project"
	confKeyBaseURL      = "base-url"
	confKeyDefaultLogin = "default-login"

	defaultTimeout = 60 * time.Second
)

var _ core.BridgeImpl = &Gitea{}

type Gitea struct{}

func (Gitea) Target() string {
	return target
}

func (g *Gitea) LoginMetaKey() string {
	return metaKeyGiteaLogin
}

func (Gitea) NewImporter() core.Importer {
	return &giteaImporter{}
}

func (Gitea) NewExporter() core.Exporter {
	return nil
	// return &giteaExporter{}
}

func buildClient(baseURL string, token *auth.Token) (*gitea.Client, error) {
	giteaClient, err := gitea.NewClient(baseURL, gitea.SetToken(token.Value))
	if err != nil {
		return nil, err
	}

	return giteaClient, nil
}
