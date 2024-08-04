package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBugTitleCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "title [BUG_ID]",
		Short:   "Display the title of a bug",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugTitle(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	cmd.AddCommand(newBugTitleEditCommand(env))

	return cmd
}

func runBugTitle(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Compile()

	env.Out.Println(snap.Title)

	return nil
}
