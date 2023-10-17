package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/repository"
)

func newPushCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "push [REMOTE]",
		Short:   "Push updates to a git remote",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runPush(env, args)
		}),
		ValidArgsFunction: completion.GitRemote(env),
	}

	return cmd
}

func runPush(env *execenv.Env, args []string) error {
	var remote string
	switch {
	case len(args) > 1:
		return errors.New("Only pushing to one remote at a time is supported")
	case len(args) == 1:
		remote = args[0]
	default:
		v, err := repository.GetDefaultString("git-bug.remote", env.Repo.AnyConfig(), "origin")
		if err != nil {
			return err
		}
		remote = v
	}

	stdout, err := env.Backend.Push(remote)
	if err != nil {
		return err
	}

	env.Out.Println(stdout)

	return nil
}
