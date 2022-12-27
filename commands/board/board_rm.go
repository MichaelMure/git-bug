package boardcmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBoardRmCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "rm BOARD_ID",
		Short:   "Remove an existing board",
		Long:    "Remove an existing board in the local repository.",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardRm(env, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	return cmd
}

func runBoardRm(env *execenv.Env, args []string) (err error) {
	if len(args) == 0 {
		return errors.New("you must provide a board prefix to remove")
	}

	err = env.Backend.Boards().Remove(args[0])

	if err != nil {
		return
	}

	env.Out.Printf("board %s removed\n", args[0])

	return
}
