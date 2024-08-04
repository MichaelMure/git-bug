package bugcmd

import (
	"github.com/spf13/cobra"

	buginput "github.com/MichaelMure/git-bug/commands/bug/input"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/util/text"
)

type bugTitleEditOptions struct {
	title          string
	nonInteractive bool
}

func newBugTitleEditCommand(env *execenv.Env) *cobra.Command {
	options := bugTitleEditOptions{}

	cmd := &cobra.Command{
		Use:     "edit [BUG_ID]",
		Short:   "Edit a title of a bug",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugTitleEdit(env, options, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue",
	)
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBugTitleEdit(env *execenv.Env, opts bugTitleEditOptions, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Compile()

	if opts.title == "" {
		if opts.nonInteractive {
			env.Err.Println("No title given. Use -m or -F option to specify a title. Aborting.")
			return nil
		}
		opts.title, err = buginput.BugTitleEditorInput(env.Repo, snap.Title)
		if err == buginput.ErrEmptyTitle {
			env.Out.Println("Empty title, aborting.")
			return nil
		}
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
