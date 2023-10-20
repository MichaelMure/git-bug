package bugcmd

import (
	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/util/colors"
)

func newBugCommentCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comment [BUG_ID]",
		Short:   "List a bug's comments",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugComment(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	cmd.AddCommand(newBugCommentNewCommand(env))
	cmd.AddCommand(newBugCommentEditCommand(env))

	return cmd
}

func runBugComment(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Compile()

	for i, comment := range snap.Comments {
		if i != 0 {
			env.Out.Println()
		}

		env.Out.Printf("Author: %s\n", colors.Magenta(comment.Author.DisplayName()))
		env.Out.Printf("Id: %s\n", colors.Cyan(comment.CombinedId().Human()))
		env.Out.Printf("Date: %s\n\n", comment.FormatTime())
		env.Out.Println(text.LeftPadLines(comment.Message, 4))
	}

	return nil
}
