package commands

import (
	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/colors"
)

func newCommentCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "comment [ID]",
		Short:    "Display or add comments to a bug.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComment(env, args)
		},
		ValidArgsFunction: completeBug(env),
	}

	cmd.AddCommand(newCommentAddCommand())

	return cmd
}

func runComment(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	for i, comment := range snap.Comments {
		if i != 0 {
			env.out.Println()
		}

		env.out.Printf("Author: %s\n", colors.Magenta(comment.Author.DisplayName()))
		env.out.Printf("Id: %s\n", colors.Cyan(comment.Id().Human()))
		env.out.Printf("Date: %s\n\n", comment.FormatTime())
		env.out.Println(text.LeftPadLines(comment.Message, 4))
	}

	return nil
}
