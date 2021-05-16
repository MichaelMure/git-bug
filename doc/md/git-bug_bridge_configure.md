## git-bug bridge configure

Configure a new bridge.

### Synopsis

	Configure a new bridge by passing flags or/and using interactive terminal prompts. You can avoid all the terminal prompts by passing all the necessary flags to configure your bridge.

```
git-bug bridge configure [flags]
```

### Examples

```
# Interactive example
[1]: github
[2]: gitlab
[3]: jira
[4]: launchpad-preview

target: 1
name [default]: default

Detected projects:
[1]: github.com/a-hilaly/git-bug
[2]: github.com/MichaelMure/git-bug

[0]: Another project

Select option: 1

[1]: user provided token
[2]: interactive token creation
Select option: 1

You can generate a new token by visiting https://github.com/settings/tokens.
Choose 'Generate new token' and set the necessary access scope for your repository.

The access scope depend on the type of repository.
Public:
	- 'public_repo': to be able to read public repositories
Private:
	- 'repo'       : to be able to read private repositories

Enter token: 87cf5c03b64029f18ea5f9ca5679daa08ccbd700
Successfully configured bridge: default

# For GitHub
git bug bridge configure \
    --name=default \
    --target=github \
    --owner=$(OWNER) \
    --project=$(PROJECT) \
    --token=$(TOKEN)

# For Launchpad
git bug bridge configure \
    --name=default \
    --target=launchpad-preview \
	--url=https://bugs.launchpad.net/ubuntu/

# For Gitlab
git bug bridge configure \
    --name=default \
    --target=github \
    --url=https://github.com/michaelmure/git-bug \
    --token=$(TOKEN)
```

### Options

```
  -n, --name string         A distinctive name to identify the bridge
  -t, --target string       The target of the bridge. Valid values are [gitea-preview,github,gitlab,jira,launchpad-preview]
  -u, --url string          The URL of the remote repository
  -b, --base-url string     The base URL of your remote issue tracker
  -l, --login string        The login on your remote issue tracker
  -c, --credential string   The identifier or prefix of an already known credential for your remote issue tracker (see "git-bug bridge auth")
      --token string        A raw authentication token for the remote issue tracker
      --token-stdin         Will read the token from stdin and ignore --token
  -o, --owner string        The owner of the remote repository
  -p, --project string      The name of the remote repository
      --non-interactive     Do not ask for user input
  -h, --help                help for configure
```

### SEE ALSO

* [git-bug bridge](git-bug_bridge.md)	 - Configure and use bridges to other bug trackers.

