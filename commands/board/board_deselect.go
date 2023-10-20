package boardcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entities/board"
)

func newBoardDeselectCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:   "deselect",
		Short: "Clear the implicitly selected board",

		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardDeselect(env)
		}),
	}

	return cmd
}

func runBoardDeselect(env *execenv.Env) error {
	err := _select.Clear(env.Backend, board.Namespace)
	if err != nil {
		return err
	}

	return nil
}
