package boardcmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/commands/input"
	"github.com/MichaelMure/git-bug/entities/board"
	"github.com/MichaelMure/git-bug/util/text"
)

type boardNewOptions struct {
	title          string
	description    string
	columns        []string
	nonInteractive bool
}

func newBoardNewCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := boardNewOptions{}

	cmd := &cobra.Command{
		Use:     "new",
		Short:   "Create a new board",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugNew(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue")
	flags.StringVarP(&options.description, "description", "d", "",
		"Provide a message to describe the board")
	flags.StringArrayVarP(&options.columns, "columns", "c", board.DefaultColumns,
		fmt.Sprintf("Define the columns of the board (default to %s)",
			strings.Join(board.DefaultColumns, ",")))
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBugNew(env *execenv.Env, opts boardNewOptions) error {
	var err error

	if !opts.nonInteractive && opts.title == "" {
		opts.title, err = input.Prompt("Board title", "title", input.Required)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.description == "" {
		opts.description, err = input.Prompt("Board description", "description")
		if err != nil {
			return err
		}
	}

	for i, column := range opts.columns {
		opts.columns[i] = text.Cleanup(column)
	}

	b, _, err := env.Backend.Boards().New(
		text.CleanupOneLine(opts.title),
		text.CleanupOneLine(opts.description),
		opts.columns,
	)
	if err != nil {
		return err
	}

	env.Out.Printf("%s created\n", b.Id().Human())

	return nil
}
