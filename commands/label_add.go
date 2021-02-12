package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/MichaelMure/git-bug/commands/select"
)

func newLabelAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "add [ID] LABEL...",
		Short:    "Add a label to a bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelAdd(env, args)
		},
		ValidArgsFunction: completeBugAndLabels(env, true),
	}

	return cmd
}

func runLabelAdd(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	added := args

	changes, _, err := b.ChangeLabels(added, nil)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
