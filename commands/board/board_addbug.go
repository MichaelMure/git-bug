package boardcmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	bugcmd "github.com/MichaelMure/git-bug/commands/bug"
	"github.com/MichaelMure/git-bug/commands/execenv"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entity"
)

type boardAddBugOptions struct {
	column string
}

func newBoardAddBugCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := boardAddBugOptions{}

	cmd := &cobra.Command{
		Use:     "add-bug [BOARD_ID] [BUG_ID]",
		Short:   "Add a bug to a board",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardAddBug(env, options, args)
		}),
		ValidArgsFunction: BoardAndBugCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.column, "column", "c", "1",
		"The column to add to. Either a column Id or prefix, or the column number starting from 1.")
	_ = cmd.RegisterFlagCompletionFunc("column", ColumnCompletion(env))

	return cmd
}

func runBoardAddBug(env *execenv.Env, opts boardAddBugOptions, args []string) error {
	board, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	var columnId entity.CombinedId

	switch {
	case err == nil:
		// try to parse as column number
		index, err := strconv.Atoi(opts.column)
		if err == nil {
			if index-1 >= 0 && index-1 < len(board.Snapshot().Columns) {
				columnId = board.Snapshot().Columns[index-1].CombinedId
			} else {
				return fmt.Errorf("invalid column")
			}
		}
		fallthrough // could be an Id
	case _select.IsErrNoValidId(err):
		board, columnId, err = env.Backend.Boards().ResolveColumn(opts.column)
		if err != nil {
			return err
		}
	default:
		// actual error
		return err
	}

	bug, _, err := bugcmd.ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	id, _, err := board.AddItemEntity(columnId, bug)
	if err != nil {
		return err
	}

	env.Out.Printf("%s created\n", id.Human())

	return board.Commit()
}
