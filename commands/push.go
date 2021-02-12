package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

func newPushCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "push [REMOTE]",
		Short:    "Push bugs update to a git remote.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(env, args)
		},
		ValidArgsFunction: completeGitRemote(env),
	}

	return cmd
}

func runPush(env *Env, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	stdout, err := env.backend.Push(remote)
	if err != nil {
		return err
	}

	env.out.Println(stdout)

	return nil
}
