## git-bug board add-draft

Add a draft item to a board

```
git-bug board add-draft [BOARD_ID] [flags]
```

### Options

```
  -t, --title string      Provide the title to describe the draft item
  -m, --message string    Provide the message of the draft item
  -F, --file string       Take the message from the given file. Use - to read the message from the standard input
  -c, --column string     The column to add to. Either a column Id or prefix, or the column number starting from 1. (default "1")
      --non-interactive   Do not ask for user input
  -h, --help              help for add-draft
```

### SEE ALSO

* [git-bug board](git-bug_board.md)	 - List boards

