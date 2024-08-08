package boardcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/cmdjson"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/entities/board"
)

type boardShowOptions struct {
	format string
}

func newBoardShowCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := boardShowOptions{}

	cmd := &cobra.Command{
		Use:     "show [BOARD_ID]",
		Short:   "Display a board",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardShow(env, options, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.format, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,json,org-mode]")

	return cmd
}

func runBoardShow(env *execenv.Env, opts boardShowOptions, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	switch opts.format {
	case "json":
		return showJsonFormatter(env, snap)
	case "default":
		return showDefaultFormatter(env, snap)
	default:
		return fmt.Errorf("unknown format %s", opts.format)
	}
}

func showDefaultFormatter(env *execenv.Env, snapshot *board.Snapshot) error {
	// // Header
	// env.Out.Printf("%s [%s] %s\n\n",
	// 	colors.Cyan(snapshot.Id().Human()),
	// 	colors.Yellow(snapshot.Status),
	// 	snapshot.Title,
	// )
	//
	// env.Out.Printf("%s opened this issue %s\n",
	// 	colors.Magenta(snapshot.Author.DisplayName()),
	// 	snapshot.CreateTime.String(),
	// )
	//
	// env.Out.Printf("This was last edited at %s\n\n",
	// 	snapshot.EditTime().String(),
	// )
	//
	// // Labels
	// var labels = make([]string, len(snapshot.Labels))
	// for i := range snapshot.Labels {
	// 	labels[i] = string(snapshot.Labels[i])
	// }
	//
	// env.Out.Printf("labels: %s\n",
	// 	strings.Join(labels, ", "),
	// )
	//
	// // Actors
	// var actors = make([]string, len(snapshot.Actors))
	// for i := range snapshot.Actors {
	// 	actors[i] = snapshot.Actors[i].DisplayName()
	// }
	//
	// env.Out.Printf("actors: %s\n",
	// 	strings.Join(actors, ", "),
	// )
	//
	// // Participants
	// var participants = make([]string, len(snapshot.Participants))
	// for i := range snapshot.Participants {
	// 	participants[i] = snapshot.Participants[i].DisplayName()
	// }
	//
	// env.Out.Printf("participants: %s\n\n",
	// 	strings.Join(participants, ", "),
	// )
	//
	// // Comments
	// indent := "  "
	//
	// for i, comment := range snapshot.Comments {
	// 	var message string
	// 	env.Out.Printf("%s%s #%d %s <%s>\n\n",
	// 		indent,
	// 		comment.CombinedId().Human(),
	// 		i,
	// 		comment.Author.DisplayName(),
	// 		comment.Author.Email(),
	// 	)
	//
	// 	if comment.Message == "" {
	// 		message = colors.BlackBold(colors.WhiteBg("No description provided."))
	// 	} else {
	// 		message = comment.Message
	// 	}
	//
	// 	env.Out.Printf("%s%s\n\n\n",
	// 		indent,
	// 		message,
	// 	)
	// }

	return nil
}

func showJsonFormatter(env *execenv.Env, snap *board.Snapshot) error {
	jsonBoard := cmdjson.NewBoardSnapshot(snap)
	return env.Out.PrintJSON(jsonBoard)
}
