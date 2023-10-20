package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBugLabelCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "label [BUG_ID]",
		Short:   "Display labels of a bug",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugLabel(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	cmd.AddCommand(newBugLabelNewCommand(env))
	cmd.AddCommand(newBugLabelRmCommand(env))

	return cmd
}

func runBugLabel(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Compile()

	for _, l := range snap.Labels {
		env.Out.Println(l)
	}

	return nil
}
