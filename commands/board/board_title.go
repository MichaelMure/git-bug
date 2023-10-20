package boardcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBoardTitleCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "title [BOARD_ID]",
		Short:   "Display the title of a board",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardTitle(env, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	cmd.AddCommand(newBoardTitleEditCommand())

	return cmd
}

func runBoardTitle(env *execenv.Env, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	env.Out.Println(snap.Title)

	return nil
}
