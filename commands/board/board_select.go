package boardcmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/execenv"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entities/board"
)

func ResolveSelected(repo *cache.RepoCache, args []string) (*cache.BoardCache, []string, error) {
	return _select.Resolve[*cache.BoardCache](repo, board.Typename, board.Namespace, repo.Boards(), args)
}

func newBoardSelectCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:   "select BOARD_ID",
		Short: "Select a board for implicit use in future commands",
		Long: `Select a board for implicit use in future commands.

The complementary command is "git board deselect" performing the opposite operation.
`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardSelect(env, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	return cmd
}

func runBoardSelect(env *execenv.Env, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a board id")
	}

	prefix := args[0]

	b, err := env.Backend.Boards().ResolvePrefix(prefix)
	if err != nil {
		return err
	}

	err = _select.Select(env.Backend, board.Namespace, b.Id())
	if err != nil {
		return err
	}

	env.Out.Printf("selected board %s: %s\n", b.Id().Human(), b.Snapshot().Title)

	return nil
}
