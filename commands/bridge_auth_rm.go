package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core/auth"
)

func newBridgeAuthRm() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "rm ID",
		Short:   "Remove a credential.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgeAuthRm(env, args)
		},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeBridgeAuth(env),
	}

	return cmd
}

func runBridgeAuthRm(env *Env, args []string) error {
	cred, err := auth.LoadWithPrefix(env.repo, args[0])
	if err != nil {
		return err
	}

	err = auth.Remove(env.repo, cred.ID())
	if err != nil {
		return err
	}

	env.out.Printf("credential %s removed\n", cred.ID())
	return nil
}
