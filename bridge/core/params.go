package core

import "fmt"

// BridgeParams holds parameters to simplify the bridge configuration without
// having to make terminal prompts.
type BridgeParams struct {
	URL        string // complete URL of a repo               (Github, Gitlab,     , Launchpad)
	BaseURL    string // base URL for self-hosted instance    (        Gitlab, Jira,          )
	Login      string // username for the passed credential   (Github, Gitlab, Jira,          )
	CredPrefix string // ID prefix of the credential to use   (Github, Gitlab, Jira,          )
	TokenRaw   string // pre-existing token to use            (Github, Gitlab,     ,          )
	Owner      string // owner of the repo                    (Github,       ,     ,          )
	Project    string // name of the repo or project key      (Github,       , Jira, Launchpad)
	Filter     string // import filter                        (      ,       . Jira,          )
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
	case "Filter":
		return fmt.Sprintf("warning: --filter is ineffective for a %s bridge", target)
	default:
		panic("unknown field")
	}
}
