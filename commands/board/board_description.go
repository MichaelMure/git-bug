package boardcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBoardDescriptionCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "description [BOARD_ID]",
		Short:   "Display the description of a board",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardDescription(env, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	cmd.AddCommand(newBoardDescriptionEditCommand())

	return cmd
}

func runBoardDescription(env *execenv.Env, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	env.Out.Println(snap.Description)

	return nil
}
