package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
)

func newBridgeRm() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm NAME",
		Short:    "Delete a configured bridge.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgeRm(env, args)
		},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeBridge(env),
	}

	return cmd
}

func runBridgeRm(env *Env, args []string) error {
	err := bridge.RemoveBridge(env.backend, args[0])
	if err != nil {
		return err
	}

	env.out.Printf("Successfully removed bridge configuration %v\n", args[0])
	return nil
}
