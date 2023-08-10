package core

import "fmt"

// BridgeParams holds parameters to simplify the bridge configuration without
// having to make terminal prompts.
type BridgeParams struct {
	URL        string // complete URL of a repo               (Gitea, Github, Gitlab,     , Launchpad)
	BaseURL    string // base URL for self-hosted instance    (     ,       , Gitlab, Jira,          )
	Login      string // username for the passed credential   (Gitea, Github, Gitlab, Jira,          )
	CredPrefix string // ID prefix of the credential to use   (Gitea, Github, Gitlab, Jira,          )
	TokenRaw   string // pre-existing token to use            (Gitea, Github, Gitlab,     ,          )
	Owner      string // owner of the repo                    (     , Github,       ,     ,          )
	Project    string // name of the repo or project key      (     , Github,       , Jira, Launchpad)
}

func (BridgeParams) fieldWarning(field string, target string) string {
	switch field {
	case "URL":
		return fmt.Sprintf("warning: --url is ineffective for a %s bridge", target)
	case "BaseURL":
		return fmt.Sprintf("warning: --base-url is ineffective for a %s bridge", target)
	case "Login":
		return fmt.Sprintf("warning: --login is ineffective for a %s bridge", target)
	case "CredPrefix":
		return fmt.Sprintf("warning: --credential is ineffective for a %s bridge", target)
	case "TokenRaw":
		return fmt.Sprintf("warning: tokens are ineffective for a %s bridge", target)
	case "Owner":
		return fmt.Sprintf("warning: --owner is ineffective for a %s bridge", target)
	case "Project":
		return fmt.Sprintf("warning: --project is ineffective for a %s bridge", target)
	default:
		panic("unknown field")
	}
}
