package boardcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/commands/input"
	"github.com/MichaelMure/git-bug/util/text"
)

type boardTitleEditOptions struct {
	title          string
	nonInteractive bool
}

func newBoardTitleEditCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := boardTitleEditOptions{}

	cmd := &cobra.Command{
		Use:     "edit [BUG_ID]",
		Short:   "Edit a title of a board",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugTitleEdit(env, options, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the board",
	)
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBugTitleEdit(env *execenv.Env, opts boardTitleEditOptions, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	if opts.title == "" {
		if opts.nonInteractive {
			env.Err.Println("No title given. Aborting.")
			return nil
		}
		opts.title, err = input.PromptDefault("Board title", "title", snap.Title, input.Required)
		if err != nil {
			return err
		}
	}

	if opts.title == snap.Title {
		env.Err.Println("No change, aborting.")
	}

	_, err = b.SetTitle(text.CleanupOneLine(opts.title))
	if err != nil {
		return err
	}

	return b.Commit()
}
