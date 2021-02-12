package commands

import (
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "status [ID]",
		Short:    "Display or change a bug status.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(env, args)
		},
		ValidArgsFunction: completeBug(env),
	}

	cmd.AddCommand(newStatusCloseCommand())
	cmd.AddCommand(newStatusOpenCommand())

	return cmd
}

func runStatus(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	env.out.Println(snap.Status)

	return nil
}
