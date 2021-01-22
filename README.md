<p align="center">
    <img width="150px" src="https://cdn.rawgit.com/MichaelMure/git-bug/master/misc/logo/logo-alpha-flat-bg.svg">
</p>
<h1 align="center">git-bug</h1>

<div align="center">

[![Build Status](https://travis-ci.com/MichaelMure/git-bug.svg?branch=master)](https://travis-ci.com/MichaelMure/git-bug)
[![Backers on Open Collective](https://opencollective.com/git-bug/backers/badge.svg)](#backers)
[![Sponsors on Open Collective](https://opencollective.com/git-bug/sponsors/badge.svg)](#sponsors)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3+-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)
[![GoDoc](https://godoc.org/github.com/MichaelMure/git-bug?status.svg)](https://godoc.org/github.com/MichaelMure/git-bug)
[![Go Report Card](https://goreportcard.com/badge/github.com/MichaelMure/git-bug)](https://goreportcard.com/report/github.com/MichaelMure/git-bug)
[![Gitter chat](https://badges.gitter.im/gitterHQ/gitter.png)](https://gitter.im/the-git-bug/Lobby)
![Docker Cloud Build Status](https://img.shields.io/docker/cloud/build/glancingmind/git-bug)

</div>

`git-bug` is a bug tracker that:

- **is fully embedded in git**: you only need your git repository to have a bug tracker
- **is distributed**: use your normal git remote to collaborate, push and pull your bugs!
- **works offline**: in a plane or under the sea? Keep reading and writing bugs!
- **prevents vendor lock-in**: your usual service is down or went bad? You already have a full backup.
- **is fast**: listing bugs or opening them is a matter of milliseconds
- **doesn't pollute your project**: no files are added in your project
- **integrates with your tooling**: use the UI you like (CLI, terminal, web) or integrate with your existing tools through the CLI or the GraphQL API
- **bridges to other bug trackers**: use [bridges](#bridges) to import and export to other trackers.

:construction: This is now more than a proof of concept, but still not fully stable. Expect dragons and unfinished business. :construction:

## Help needed!

This project has grown bigger than I can handle by myself, especially with a day job. I'm looking for people to help on or maintain part of it:
- each bridges (go)
- the terminal UI (go)
- the web UI (Typescript/React/GraphQL)

Individually, those pieces are not especially complex but doing everything make it hard for me to focus on the core where things get more complicated. If this is useful for you or you just want to join the fun, maybe consider it?

## Installation

<details><summary>Pre-compiled binaries</summary>

1. Go to the [release page](https://github.com/MichaelMure/git-bug/releases/latest) and download the appropriate binary for your system.
2. Copy the binary anywhere in your $PATH
3. Rename the binary to `git-bug` (or `git-bug.exe` on windows)

That's all !

</details>

<details><summary>Container Image</summary>

  The latest build can be pulled as container image from [dockerhub](https://hub.docker.com/r/glancingmind/git-bug) via: `docker pull glancingmind/git-bug`
</details>

<details><summary>Linux packages</summary>

* [Archlinux (AUR)](https://aur.archlinux.org/packages/?K=git-bug)
* [NixOS](https://github.com/NixOS/nixpkgs/blob/master/pkgs/applications/version-management/git-and-tools/git-bug/default.nix#L31)

</details>

<details><summary>macOS packages</summary>

* [Homebrew](https://formulae.brew.sh/formula/git-bug)
  ```
  brew install git-bug
  ```

</details>

<details><summary>FreeBSD package/port</summary>

Install via the package manager
  ```
  pkg install git-bug
  ```

Or from the ports collection
  ```
  cd /usr/ports/devel/git-bug && make install clean
  ```

</details>

<details><summary>Compile from git (unstable)</summary>

```shell
git clone git@github.com:MichaelMure/git-bug.git
make install
```

If it's not done already, add the golang binary directory in your PATH:

```bash
export PATH=$PATH:$(go env GOROOT)/bin:$(go env GOPATH)/bin
```

</details>

## Workflows

There are multiple ways to use `git-bug`:

<details><summary>Native workflow</summary>
<p align="center">
    <img src="misc/diagrams/native_workflow.png" alt="Native workflow">
</p>

This is the pure `git-bug` experience. In a similar fashion as with code, use `git bug push` and `git bug pull` to push and pull your bugs between git remotes and collaborate with your teammate. 

</details>

<details><summary>Bridge workflow</summary>
<p align="center">
    <img src="misc/diagrams/bridge_workflow.png" alt="Bridge workflow">
</p>

As `git-bug` has bridges with other bug-trackers, you can use it as your personal local remote interface. Sync with `git bug bridge pull` and `git bug bridge push`, work from your terminal, integrate into your editor, it's up to you. And it works offline !

</details>

<details><summary>Web UI workflow (WIP)</summary>
<p align="center">
    <img src="misc/diagrams/webui-workflow.png" alt="Web UI workflow">
</p>

Often, projects needs to have their bug-tracker public and accept editions from anyone facing a problem. To support this workflow, `git-bug` aims to have the web UI accept external OAuth authentication and act as a public portal. However the web UI is not up to speed for that yet. Contribution are very much welcome!

</details>

## CLI usage

Create a new identity:

```
git bug user create
```

Create a new bug:

```
git bug add
```

Your favorite editor will open to write a title and a message.

You can push your new entry to a remote:
```
git bug push [<remote>]
```

And pull for updates:
```
git bug pull [<remote>]
```

List existing bugs:
```
git bug ls
```

Filter and sort bugs using a [query](doc/queries.md):
```
git bug ls "status:open sort:edit"
```

Search for bugs by text content:
```
git bug ls "foo bar" baz
```

You can now use commands like `show`, `comment`, `open` or `close` to display and modify bugs. For more details about each command, you can run `git bug <command> --help` or read the [command's documentation](doc/md/git-bug.md).

## Interactive terminal UI

An interactive terminal UI is available using the command `git bug termui` to browse and edit bugs.

![Termui recording](misc/termui_recording.gif)

## Web UI (status: WIP)

You can launch a rich Web UI with `git bug webui`.

<p align="center">
  <img src="misc/webui1.png" alt="Web UI screenshot 1" width="880">
</p>

<p align="center">
  <img src="misc/webui2.png" alt="Web UI screenshot 2" width="880">
</p>

This web UI is entirely packed inside the same go binary and serve static content through a localhost http server.

The web UI interact with the backend through a GraphQL API. The schema is available [here](api/graphql/schema).

## Container Image

The container functions like the git-bug binary (with some small exception).  
Just append the corresponding cli-commands at the end of the container invokation:
```shell
docker run -i -t glancingmind/git-bug <cli-command>
```
E.g.
- Show help: `docker run -i -t glancingmind/git-bug --help`
- Open terminalui: `docker run -i -t glancingmind/git-bug termui`

### Small exceptions

- For your convenience, the webui is started on port 3000, when no cli-command is given.
  E.g.
  
  ```shell
  docker run -i -t glancingmind/git-bug
  ```

- Also the container image includes a preconfigured issue-repository. 
  This is your own personal and ready-to-go playground. Horay!
  But if you want to use your own issue-repository, then mount it inside the container at the `/repository` path. E.g.
  ```shell
  docker run -i -t --rm -v <path-to-your-issue-repository>:/repository glancingmind/git-bug
  ```
  And don't forget to create a git-bug identity, as describted [here](https://github.com/GlancingMind/git-bug#cli-usage).
  **NOTE** The user name and email of the Issue-repositories TestUser can be
  overwritten by setting the ISSUE_REPO_USER_NAME and ISSUE_REPO_USER_EMAIL
  as build arguments on container build.

## Bridges

### Importer implementations

|                                                 | Github             | Gitlab             | Jira               | Launchpad          |
|-------------------------------------------------|:------------------:|:------------------:|:------------------:|:------------------:|
| **incremental**<br/>(can import more than once) | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| **with resume**<br/>(download only new data)    | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| **identities**                                  | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: |
| identities update                               | :x:                | :x:                | :x:                | :x:                |
| **bug**                                         | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: |
| comments                                        | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: |
| comment editions                                | :heavy_check_mark: | :x:                | :heavy_check_mark: | :x:                |
| labels                                          | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| status                                          | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| title edition                                   | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:                |
| **media/files**                                 | :x:                | :x:                | :x:                | :x:                |
| **automated test suite**                        | :heavy_check_mark: | :heavy_check_mark: | :x:                | :x:                |

### Exporter implementations

|                          | Github             | Gitlab             | Jira               | Launchpad |
|--------------------------|:------------------:|:------------------:|:------------------:|:---------:|
| **bug**                  | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| comments                 | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| comment editions         | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| labels                   | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| status                   | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| title edition            | :heavy_check_mark: | :heavy_check_mark: | :heavy_check_mark: | :x:       |
| **automated test suite** | :heavy_check_mark: | :heavy_check_mark: | :x:                | :x:       |

#### Bridge usage

Interactively configure a new github bridge:

```bash
git bug bridge configure
```

Or manually:

```bash
git bug bridge configure \
    --name=<bridge> \
    --target=github \
    --url=https://github.com/MichaelMure/git-bug \
    --login=<login>
    --token=<token>
```

Import bugs:

```bash
git bug bridge pull [<name>]
```

Export modifications:

```bash
git bug bridge push [<name>]
```

Deleting a bridge:

```bash
git bug bridge rm [<name>]
```

## Internals

Interested by how it works ? Have a look at the [data model](doc/model.md) and the [internal bird-view](doc/architecture.md).

## Misc

- [Bash completion](misc/bash_completion)
- [Zsh completion](misc/zsh_completion)
- [PowerShell completion](misc/powershell_completion)
- [ManPages](doc/man)

## Planned features

- media embedding
- more bridges
- extendable data model to support arbitrary bug tracker
- inflatable raptor

## Contribute

PRs accepted. Drop by the [Gitter lobby](https://gitter.im/the-git-bug/Lobby) for a chat or browse the issues to see what is worked on or discussed.

```shell
git clone git@github.com:MichaelMure/git-bug.git
```

You can now run `make` to build the project, or `make install` to install the binary in `$GOPATH/bin/`.

Alternativly `docker build .` can be invoked from the root directory to build a container image.
But be aware, that this will make a release build. Advantage, it is no longer required to have go installed on the system.

To work on the web UI, have a look at [the dedicated Readme.](webui/Readme.md)


## Contributors :heart:

This project exists thanks to all the people who contribute.
<a href="https://github.com/MichaelMure/git-bug/graphs/contributors"><img src="https://opencollective.com/git-bug/contributors.svg?width=890&button=false" /></a>


## Backers

Thank you to all our backers! 🙏 [[Become a backer](https://opencollective.com/git-bug#backer)]

<a href="https://opencollective.com/git-bug#backers" target="_blank"><img src="https://opencollective.com/git-bug/tiers/backer.svg?width=890"></a>


## Sponsors

Support this project by becoming a sponsor. Your logo will show up here with a link to your website. [[Become a sponsor](https://opencollective.com/git-bug#sponsor)]

<a href="https://opencollective.com/git-bug/sponsor/0/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/1/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/2/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/3/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/4/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/5/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/6/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/7/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/8/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/9/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/9/avatar.svg"></a>


## License

Unless otherwise stated, this project is released under the [GPLv3](LICENSE) or later license © Michael Muré.

The git-bug logo by [Viktor Teplov](https://github.com/vandesign) is released under the [Creative Commons Attribution 4.0 International (CC BY 4.0)](misc/logo/LICENSE) license © Viktor Teplov.
