## git-bug export

Export bugs as a series of operations.

### Synopsis

Export bugs as a series of operations.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language, flags, a natural language full text search, or a combination of the aforementioned.

```
git-bug export [QUERY] [flags]
```

### Examples

```
See ls
```

### Options

```
  -s, --status strings        Filter by status. Valid values are [open,closed]
  -a, --author strings        Filter by author
  -p, --participant strings   Filter by participant
  -A, --actor strings         Filter by actor
  -l, --label strings         Filter by label
  -t, --title strings         Filter by title
  -n, --no strings            Filter by absence of something. Valid values are [label]
  -b, --by string             Sort the results by a characteristic. Valid values are [id,creation,edit] (default "creation")
  -d, --direction string      Select the sorting direction. Valid values are [asc,desc] (default "asc")
  -h, --help                  help for export
```

### SEE ALSO

* [git-bug](git-bug.md)	 - A bug tracker embedded in Git.

