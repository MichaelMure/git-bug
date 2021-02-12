package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/input"
)

type titleEditOptions struct {
	title string
}

func newTitleEditCommand() *cobra.Command {
	env := newEnv()
	options := titleEditOptions{}

	cmd := &cobra.Command{
		Use:      "edit [ID]",
		Short:    "Edit a title of a bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTitleEdit(env, options, args)
		},
		ValidArgsFunction: completeBug(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue",
	)

	return cmd
}

func runTitleEdit(env *Env, opts titleEditOptions, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	if opts.title == "" {
		opts.title, err = input.BugTitleEditorInput(env.repo, snap.Title)
		if err == input.ErrEmptyTitle {
			env.out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	if opts.title == snap.Title {
		env.err.Println("No change, aborting.")
	}

	_, err = b.SetTitle(opts.title)
	if err != nil {
		return err
	}

	return b.Commit()
}
