# Selectively Bridge Bugs

## Motivation

I am currently using git-bug to manage my bug/feature/task list for a number of
projects. Some of those projects have "external" bug trackers (e.g. my day job,
or github mirrors of side projects).

My personal bug repo covers 20 projects. I prefer the experience of interacting
with a single git-bug session in `termui` versus a half-dozen different
laggy-ass hyper-ajax web 2.0 online trackers.

I would love to sync my personal bug repo with these external trackers but there
are a couple of problems:

1. Our JIRA at work has millions of bugs. I obviously don't interact with
   all of them. I only want to sync bugs that are "relevant" to me... where
   "relevant" can presumably be defined by matching some criteria on metadata
   in JIRA.
2. Some of the bugs in my repo really shouldn't be pushed to a public location
   because they might contain "sensitive" information about my employer.
3. I don't want bugs for irrelevant projects to pollute the issue list on
   my github mirrors.

## Proposed Solution

In order to address the issues, I propose the following new features:

1. Filter bugs during import so that only a subset of the remote bugs are
   considered for import into `git-bug`.
2. Add an additional field to `git-bug` called `project`. A project can
   optionally be assigned to exactly one remote bugtracker.
3. Add an additional field to `git-bug`s called `private` which, if true,
   means that the bug should not be exported by the bridge associated with
   its project.

### Filter bugs on import

A filter specification can be provided in the bridge configuration section of
the config file. The filter specification is provided to the bridge which
translates it (to the extent possible) into query parameters for the API of the
remote bugtracker. Any filter which cannot be satisfied by query parameters is
used to construct a `git-bug` filter object, which is then applied to all bugs
returned by the remote bug tracker, preventing them from being imported into
`git-bug`.

### project metadata

Each bug may (optionally) be assiged to a `project`. When a bridge is configured
it may also be mapped to any `project` which is not yet mapped to another bridge
(it may also create a new `project`). When bugs are imported over this bridge,
they will automatically be assigned to the mapped `project`. When a `bridge
export` is executed, only bugs assigned to the mapped `project` will be
considered for export.

If only one bridge is configured and no projects are mapped to it, then it is
considered as mapped to the default project. Thus the current behavior will be
unchanged. Specifically, if a single bridge is configured all bugs will be
exported over that bridge by default.

**open question**: Should `project` be an actual explicit piece of metadata, or
should it be implicit based on the presence of remote bugtracker metadata. From
a UI perspective `git-bug` could *preset* remote tracker metadata as if it were
a `project` assignment. A user can assign a bug to a `project` in UI, where
under the hood `git-bug` simply stores the associated remote tracker metadata /
link info. My preference, however, is to keep them as separate entities so that
`git-bug` created bugs may be assigned to a "project" before a bridge is
configured for that project.

**open question**: Should it be possible to change a bug's `project` after
creation? Seems like it would be important to do so, though things might get
tricky if it has already been exported. For instance if a bug exported over
`bridge A`, and then the project assignment of that bug is changed to project
`B`, it might get recreated on the next import from `bridge A` if we're not
careful.

### private metadata

Any bug which originated outside of `git-bug` (i.e. was `import`ed) is
permanently `private=false`. Any bug which originates inside of `git-bug` is
initially created with the `private` field set to a value defined by the
configuration. If no configuration value is specified then default will be
`false`. This will match the current behavior meaning bugs created in `git-bug`
will be exported over and associated with the first bridge on which `export` is
executed.

**open question**: should the default privacy of a bug be configured only
globally, or on a per-project basis? If per-project, should a bug's privacy be
set to that value immediately on assigment to the project? My feeling is that
making it global is easier to implement and matches my desired usecase better
but that making it per-project is probably the "right" answer.

## Case Study

Consider the project `git-bug-test` which has a github mirror at
`cheshirekow.com/git-bug-test`. Ideally we would specify in our git-config
something like this:

```config
[git-bug]
  new-bug-private = true
[git-bug "bridge.github.github-git-bug-test"]
	user = cheshirekow
	project = git-bug-test
	token = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	import-filter="assignee=cheshirekow"
```

with the following semantics:

**import**: Only issues matching `assignee=cheshirekow` will be queried from
github. Any issue that is created is assigned to `project=git-bug-test`. Also
the issue is marked `private=False (permanent)`.

**export**: Only bugs which have `project=git-bug-test` and `private=False`
will be considered for export.


## User Experience of project assignment

In general, copy the UX for `labels` except that the assignment is unique and
not multiple. Possibly in some cases it cannot be changed (i.e imported bugs).

### Command line

**open questions**: Should project assignment be basically the same as label
assignment?

* `git bug ls-project`
* `git bug project <id> [<project>]`

This begs the question, what happens as more and more metadata are added? Some
metadata examples:

* priority
* backlog order
* due-date
* assignee
* subscribers
* workflow state (in-progress, in-review, resolved, closed)
* sprint
* resolution (fixed, not-a-bug, deferred)

should we plan ahead and do something more like

* `git bug ls-meta project`
* `git bug set project <id> [<project>]`
* `git bug get project <id>`

assuming that `project` could be swapped out for any metadata like `label`,
`title`, `status`, `assignee`, `resolution`, etc. in the future?

### termui

Maybe replicate the "labels" display in the bug view:

```
                                                          ┌────────────────────┐
 [e767d6f] [git-bug] export/import by label               │Project             │
                                                          │  git-bug           │
 [open] Josh Bialkowski opened this bug on Nov 1 2019     └────────────────────┘

     1. On import, assign labels to any new bugs          ┌────────────────────┐
     2. On export, only send bugs that match labels       │Labels              |
     3. Store the label list in the config                │  feature-request   |
                                                          │  contribution      |
                                                          └────────────────────┘
 Josh Bialkowski added "git-bug" label on Nov 1 2019
```

And setting the project will have the same UI as setting labels (except that
perhaps only one can be selected):

```
     [ ] argue

     [ ] buntstrap

     [ ] cmake-format

     [ ] flow-tools

     [ ] fusebus
   ┌─────────────────────┐
   │ [x] git-bug         │
   └─────────────────────┘
     [ ] oauthsub

     [ ] sphinx

     [ ] sphinx-codefence

     [ ] uchroot

```

**feature request** Ok, so this is definitely a tangent, but it would be awesome
if the label (and `project` if we go this route) UI in `termui` had autocomplete
so you could start typing the name of an entry and that entry would get selected
(instead of using arrow keys to navigate).

### webui

Again, probably follow the same UI style as labels, but in a separate `<div>`

## User Experience of privacy

In general, copy the UX for `status`

### command line

Same consideration as `project`. We can create a top level
`git bug private <id> [value]` bug maybe `git bug set private <id> [value]`
is a little more future proof?

### termui

Roughly try to mimic the `status` field. Maybe put it on a second line
like this:

```
 [e767d6f] [git-bug] export/import by label

 [open] Josh Bialkowski opened this bug on Nov 1 2019
 [private]

     1. On import, assign labels to any new bugs
     2. On export, only send bugs that match labels
     3. Store the label list in the config
```

State can be toggled with `p`:

```
[q] Save and return [←↓↑→,hjkl] Navigation [o] Toggle open/close [e] Edit [c] Comment [t] Change title [p] Toggle private
```

### webui

Not sure here, probably render similar to the `status` field.


## Implementation notes (github)

### Filtering github issues on input

The main loop-over-bugs is in `github/import.go:51`. It uses the
`githubImporter.iterator` to step through bugs. The iterator is implemented in
`github/iterator.go`. It uses the `github.com/shurcooL/githubv4` package. I
think the query is composed of the `variables["xxx"] =` assignments starting on
line `github/iterator.go:102` through line `134`. I think this is where we add
additional query parameters.

#### Filtering at query time

The [github api documentation][1] describes the GraphQL schema.

I think the query starts with an `issueTimelineQuery` which, I think, returns
issues in timeline order. In the graphQL schema it looks like the root query is
[`repository`][2] and it's [`issues`][3] connection to [`Issue`][4] objects.

[1]: https://developer.github.com/v4/object/issue/
[2]: https://developer.github.com/v4/object/repository/
[3]: https://developer.github.com/v4/object/issueconnection/
[4]: https://developer.github.com/v4/object/issue/

#### Filtering during import

The main iterator interface is `NextIssue` at `github/iterator.go:172`. I don't
fully follow what's happening here but the issue is actually retrieved by
`queryIssue(()` at `github/iterator.go:153` and it appears that `NextIssue`
returns true if the iterator has an issue cached (and it may also perform a
query to fetch and an issue if needed).

The issue itself is retrieved via `IssueValue()` at `github/iterator.go:203`,
which just returns `i.timeline.query.Repository.Issues.Nodes[0]`.

I think the easiest thing to do is to modify `github.iterator`, moving
`NextIssue` deeper down the stack and add a wrapper `NextFilteredIssue`.
`NextFilteredIssue` will call next issue, and then inspect the issue against
filters. If it is accepted it returns, like `NextIssue` would have. Otherwise,
it continues to iterate over `NextIssue` until either the issues run out or we
find the next one that matches.

### Assigning project on import

The creation of bugs in `git-bug` I think happens within `ensureIssue`, defined
at `import.go:87`. This in turn calls `cache.RepoCache.ResolveBugCreateMetadata`
defined at `cache/repo_cache.go:450`. `ResolveBug` returns the bug if it exists,
otherwise returns an error. The createion of a new bug appears to be done by
`cache.NewBugRaw`. I think the best place to add modification logic is at the
end of `ensureIssue`. `ensureIssue()` returns the bug that was either found or
newly created. Right before returning it we could make any necessary
modifications, adding or removing labels, etc.

### Filtering bugs on export

I think the main function for exporting bugs is `ExportAll()` in
`github/export.go:82`. The iteration I think happens on on line 123:

```
		allBugsIds := repo.AllBugsIds()

		for _, id := range allBugsIds {
```

The actual export appears to go through `exportBug()` at line 152. `exportBug`
is defined at line 162. There already appears to be some filtering. On line
`github/export.go:177` we see:

```
	// skip bug if origin is not allowed
	origin, ok := snapshot.GetCreateMetadata(keyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
```

`GetCreateMetadata` appears to be a dictionary lookup, where the key is added if
it doesn't exist. This looks like it wont push bugs that were pulled from a
different github repository.

This seems like a good place to check the project assignment and privacy label.
So we'll go from one check to three checks: The bug

1. was not imported by another bridge
2. is assigned to this project
3. is public

We can just return if the bug doesn't match the criteria for that remote.

### Adding config options

I think the configuration options are managed internally through the
`core.Configuration` object that gets passed to the importers and exporters. The
github exporter, for example, stores this as it's `.conf` member. The object
seems to act like a dictionary with array accessor. It seems that
`ge.conf[keyToken]` is used a few places throughout the exporter. `keyToken` is
a const defined as `"token"` in `config.go` so I think we just access the config
as a dictionary. Presumably if we add the config options above we would just
access them as `ge.conf["import-filter"]` (though probably use a constant to aid
static analysis).

### Filter and modification logic

There already exists a query/filter class within `git-bug`. We can take
advantage of this existing functionality, with the filter specification coming
from a string in the git-config. Ideally we can expose a more convenient
intermediate representation in order for bridge implementations to convert the
specification into query parameters.

From [@MichaelMure][6]:

    At the moment, the query implementation is a single process translating
    something like `status:open author:descartes sort:edit-asc` into a `Query`,
    composed of `Filter` plus what is needed to order the result. `Filter` is
		basically a `func(*BugExcerpt) bool`, that is a function matching or not a
		bug.

		To achieve what you want here, this process will need to be broken into two
		stages:

		* some kind of micro parser, translating the string into an intermediary
			representation in memory
		* a set of simple "compilers", translating that result into a Filter to act
		  onBugExcerpt or generating the corresponding parameters/requests for each
			bridge.

		Note that if a query can't be entirely translated into request parameters, a
		`Filter` as we have now can be applied to further filter before writing bugs
		in git.

[6]: https://github.com/MichaelMure/git-bug/pull/241/files#r341809118


### Adding new metadata

**open questions:**

1. How do we represent new metadata in `git-bug`?
2. Do we need  to modify code a the core/cache level?
3. How do we add a new filter key for `project:` and `public:`?

... time to brush up on my `golang`.
