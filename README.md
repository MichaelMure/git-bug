<div align="center">
<img width="64px" src="https://cdn.rawgit.com/MichaelMure/git-bug/master/misc/logo/logo-alpha-flat-bg.svg">
<h1 align="center">git-bug</h1>

[![trunk][ci/badge]][ci/url]

[![Backers on Open Collective][oc/backers/badge]](#backers)
[![Sponsors on Open Collective][oc/sponsors/badge]](#sponsors)
[![GPL v3 License][license/badge]][license/url]
[![GoDoc][godoc/badge]][godoc/url]
[![Go Report Card][report-card/badge]][report-card/url]
[![Matrix][matrix/badge]][matrix/url]

</div>

`git-bug` is a bug tracker that:

- **is fully embedded in git**: you only need your git repository to have a bug
  tracker
- **is distributed**: use your normal git remote to collaborate, push and pull
  your bugs!
- **works offline**: no internet connection? Keep reading and writing bugs!
- **prevents vendor lock-in**: is your usual service down? Good thing your bugs
  are distributed!
- **is fast**: listing bugs or opening them is a matter of milliseconds
- **doesn't pollute your project**: no files are added in your project
- **integrates with your tooling**: use the UI you like (CLI, TUI, web) or
  integrate with your existing tools through the CLI or the GraphQL API
- **bridges to other bug trackers**: use [bridges](#bridges) to import and
  export to other trackers.

---

<div align="center"><em>
This project and its vibrant community was initially dreamt up and built

by [Michael Muré][mm]. Thank you for all of your hard work!

:heart:
</em></div>

---

## Getting started

There are a few different ways to get involved.

- Check out the [Installation Guide][doc/install] to install `git-bug`
  - _Looking for the [latest release][rel/latest]?_
- Take a look at the [User Guide][doc/user-guide] for usage instructions
- Read the [Engineering Documentation][doc/contrib] and contribute if
  inspiration strikes! We welcome all programmers, new and experienced alike
- Start a [new discussion][discuss/new] or [browse existing ones][discuss] to
  ask a question or see what the community is focused on
- Join us [on Matrix][matrix/url] for ongoing, asynchronous chat

## Workflows

There are multiple ways to use `git-bug`:

<details><summary>Native workflow</summary>
<p align="center">
    <img src="misc/diagrams/native_workflow.png" alt="Native workflow">
</p>

This is the pure `git-bug` experience. In a similar fashion as with code, use
`git bug push` and `git bug pull` to push and pull your bugs between git remotes
and collaborate with your teammate.

</details>

<details><summary>Bridge workflow</summary>
<p align="center">
    <img src="misc/diagrams/bridge_workflow.png" alt="Bridge workflow">
</p>

As `git-bug` has bridges with other bug-trackers, you can use it as your
personal local remote interface. Sync with `git bug bridge pull` and `git bug
bridge push`, work from your terminal, integrate into your editor, it's up to
you. And it works offline !

</details>

<details><summary>Web UI workflow (WIP)</summary>
<p align="center">
    <img src="misc/diagrams/webui-workflow.png" alt="Web UI workflow">
</p>

Often, projects needs to have their bug-tracker public and accept editions from
anyone facing a problem. To support this workflow, `git-bug` aims to have the
web UI accept external OAuth authentication and act as a public portal. However
the web UI is not up to speed for that yet. Contribution are very much welcome!

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

You can now use commands like `show`, `comment`, `open` or `close` to display
and modify bugs. For more details about each command, you can run `git bug
<command> --help` or read the [command's documentation](doc/md/git-bug.md).

## Interactive TUI (terminal UI)

An interactive TUI (terminal UI) is available using the command `git bug termui` to
browse and edit bugs.

![Termui recording](misc/termui_recording.gif)

[api/gql/schema]: ./api/graphql/schema

## Web UI

You can launch a Web UI with `git bug webui`:

<details><summary>View a feed of bugs</summary>
<p align="center">
  <img src="misc/webui1.png" alt="Web UI screenshot 1" width="880">
</p>
</details>

<details><summary>View comments on bug</summary>
<p align="center">
  <img src="misc/webui2.png" alt="Web UI screenshot 2" width="880">
</p>
</details>

This web UI is packed inside the same binary and serves static
content through an http server running on the local host machine.

The web UI interacts with the backend through a GraphQL API. [View the
schema][api/gql/schema] for more information.

## <a name="bridges"></a> Bridge compatibility

> [!NOTE]
> Legend for the tables below:
> - :white_check_mark: _implemented_
> - :large_orange_diamond: _partial implementation_
> - :x: _not yet implemented_

### Importer implementations

|                          | Github                 | Gitlab                 | Jira                   | Launchpad              |
| ------------------------ | ---------------------- | ---------------------- | ---------------------- | ---------------------- |
| **incremental**          | :white_check_mark:     | :white_check_mark:     | :white_check_mark:     | :x:                    |
| **with resume**          | :white_check_mark:     | :white_check_mark:     | :white_check_mark:     | :x:                    |
| **identities**           | :large_orange_diamond: | :large_orange_diamond: | :large_orange_diamond: | :large_orange_diamond: |
| **bugs**                 | :white_check_mark:     | :white_check_mark:     | :white_check_mark:     | :large_orange_diamond: |
| **board**                | :x:                    | :x:                    | :x:                    | :x:                    |
| **media/files**          | :x:                    | :x:                    | :x:                    | :x:                    |
| **automated test suite** | :white_check_mark:     | :white_check_mark:     | :x:                    | :x:                    |

### Exporter implementations

|                          | Github                 | Gitlab                 | Jira                   | Launchpad              |
| ------------------------ | ---------------------- | ---------------------- | ---------------------- | ---------------------- |
| **identities**           | :large_orange_diamond: | :large_orange_diamond: | :large_orange_diamond: | :large_orange_diamond: |
| **bug**                  | :white_check_mark:     | :white_check_mark:     | :white_check_mark:     | :x:                    |
| **board**                | :x:                    | :x:                    | :x:                    | :x:                    |
| **automated test suite** | :white_check_mark:     | :white_check_mark:     | :x:                    | :x:                    |

#### Bridge usage

Interactively configure a new github bridge:

```bash
git bug bridge new
```

Or manually:

```bash
git bug bridge new \
    --name=<bridge> \
    --target=github \
    --url=https://github.com/MichaelMure/git-bug \
    --login=<login> \
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

Interested in how it works? Take a look at the [data model](doc/model.md) and
the [internal architecture](doc/architecture.md).

Or maybe you want to build your own [distributed data-structure in
git](entity/dag/example_test.go)?

[Read the documentation](doc) for more information.

## Misc

- [Shell completion](misc/completion) for Bash, Zsh, Fish, and Powershell
- View the raw [manpages](doc/man), or `man git-bug`

## Planned features

The [feature matrix](doc/feature_matrix.md) gives a good overview of what is
planned, without being exhaustive.

Additional planned feature:

- webUI that can be used as a public portal to accept user's input
- inflatable raptor

## Contribute

We welcome PRs! Drop by [Matrix][matrix/url] or for a chat, look at the [feature
matrix](doc/feature_matrix.md) or browse the issues to see what is being worked
on or discussed.

```shell
git clone git@github.com:MichaelMure/git-bug.git
```

You can now run `make` to build the project, or `make install` to install the
binary in `$GOPATH/bin/`.

To work on the web UI, have a look at [the dedicated Readme.](webui/Readme.md)

Some tests for the CLI use golden files, if the output of CLI commands is
changed, run the following command, then inspect the changed files in
`commands/testdata/...` to make sure the output text is as expected:

```shell
go test ./commands -update
```

## <a name="contributors"></a> Contributors :computer:

This project exists thanks to all of the engineering talent that has contributed
to it over the years. We couldn't do it [without your help][contributors/url]!

[![Contributors][contributors]][contributors/url]

## <a name="backers"></a> Backers :star:

Thank you to all of our backers! :tada: [Want your picture to show up here?][oc/backers/url]

<center>

[![Backers][oc/backers]][oc/backers/url]

</center>

## <a name="sponsors"></a> Sponsors :star2:

Support this project by becoming a sponsor. Your logo will show up here with a
link to your website. [Become a sponsor today!][oc/sponsors/url]

<center>

[![OC Sponsor 0][oc/sponsor/0]][oc/sponsor/0/url]
[![OC Sponsor 1][oc/sponsor/1]][oc/sponsor/1/url]
[![OC Sponsor 2][oc/sponsor/2]][oc/sponsor/2/url]
[![OC Sponsor 3][oc/sponsor/3]][oc/sponsor/3/url]
[![OC Sponsor 4][oc/sponsor/4]][oc/sponsor/4/url]
[![OC Sponsor 5][oc/sponsor/5]][oc/sponsor/5/url]
[![OC Sponsor 6][oc/sponsor/6]][oc/sponsor/6/url]
[![OC Sponsor 7][oc/sponsor/7]][oc/sponsor/7/url]
[![OC Sponsor 8][oc/sponsor/8]][oc/sponsor/8/url]
[![OC Sponsor 9][oc/sponsor/9]][oc/sponsor/9/url]

<center>

## License

Unless otherwise stated, this project and all assets within it are released
under the [GPLv3][license/url] or later license &copy; Michael Muré.

The `git-bug` logo is authored by [Viktor Teplov](https://github.com/vandesign)
and is released under the [Creative Commons Attribution 4.0 International (CC BY
4.0)](misc/logo/LICENSE) license &copy; Viktor Teplov.

[ci/badge]: https://github.com/MichaelMure/git-bug/actions/workflows/trunk.yml/badge.svg
[ci/url]: https://github.com/MichaelMure/git-bug/actions/workflows/trunk.yml
[doc/install]: INSTALLATION.md
[doc/contrib]: CONTRIBUTING.md
[doc/user-guide]: doc/user-guide/README.md
[contributors/url]: https://github.com/MichaelMure/git-bug/graphs/contributors
[contributors]: https://opencollective.com/git-bug/contributors.svg?avatarHeight=40&width=890&button=false
[godoc/badge]: https://godoc.org/github.com/MichaelMure/git-bug?status.svg
[godoc/url]: https://godoc.org/github.com/MichaelMure/git-bug
[license/badge]: https://img.shields.io/badge/License-GPLv3+-blue.svg
[license/url]: https://github.com/MichaelMure/git-bug/blob/master/LICENSE
[matrix/badge]: https://img.shields.io/badge/chat%20on%20matrix-%23238636
[matrix/url]: https://matrix.to/#/#git-bug:matrix.org
[mm]: https://github.com/MichaelMure
[oc/backers/badge]: https://opencollective.com/git-bug/backers/badge.svg
[oc/backers/url]: https://opencollective.com/git-bug#backers
[oc/backers]: https://opencollective.com/shields/backers.svg?avatarHeight=40&width=890&button=false
[oc/sponsor/0/url]: https://opencollective.com/git-bug/sponsor/0/website
[oc/sponsor/0]: https://opencollective.com/git-bug/tiers/sponsor/0/avatar.svg
[oc/sponsor/1/url]: https://opencollective.com/git-bug/sponsor/1/website
[oc/sponsor/1]: https://opencollective.com/git-bug/tiers/sponsor/1/avatar.svg
[oc/sponsor/2/url]: https://opencollective.com/git-bug/sponsor/2/website
[oc/sponsor/2]: https://opencollective.com/git-bug/tiers/sponsor/2/avatar.svg
[oc/sponsor/3/url]: https://opencollective.com/git-bug/sponsor/3/website
[oc/sponsor/3]: https://opencollective.com/git-bug/tiers/sponsor/3/avatar.svg
[oc/sponsor/4/url]: https://opencollective.com/git-bug/sponsor/4/website
[oc/sponsor/4]: https://opencollective.com/git-bug/tiers/sponsor/4/avatar.svg
[oc/sponsor/5/url]: https://opencollective.com/git-bug/sponsor/5/website
[oc/sponsor/5]: https://opencollective.com/git-bug/tiers/sponsor/5/avatar.svg
[oc/sponsor/6/url]: https://opencollective.com/git-bug/sponsor/6/website
[oc/sponsor/6]: https://opencollective.com/git-bug/tiers/sponsor/6/avatar.svg
[oc/sponsor/7/url]: https://opencollective.com/git-bug/sponsor/7/website
[oc/sponsor/7]: https://opencollective.com/git-bug/tiers/sponsor/7/avatar.svg
[oc/sponsor/8/url]: https://opencollective.com/git-bug/sponsor/8/website
[oc/sponsor/8]: https://opencollective.com/git-bug/tiers/sponsor/8/avatar.svg
[oc/sponsor/9/url]: https://opencollective.com/git-bug/sponsor/9/website
[oc/sponsor/9]: https://opencollective.com/git-bug/tiers/sponsor/9/avatar.svg
[oc/sponsors/badge]: https://opencollective.com/git-bug/sponsors/badge.svg
[oc/sponsors/url]: https://opencollective.com/git-bug#sponsor
[rel/latest]: https://github.com/MichaelMure/git-bug/releases/latest
[report-card/badge]: https://goreportcard.com/badge/github.com/MichaelMure/git-bug
[report-card/url]: https://goreportcard.com/report/github.com/MichaelMure/git-bug
