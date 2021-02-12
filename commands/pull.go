package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/entity"
)

func newPullCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "pull [REMOTE]",
		Short:    "Pull bugs update from a git remote.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(env, args)
		},
		ValidArgsFunction: completeGitRemote(env),
	}

	return cmd
}

func runPull(env *Env, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pulling from one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	env.out.Println("Fetching remote ...")

	stdout, err := env.backend.Fetch(remote)
	if err != nil {
		return err
	}

	env.out.Println(stdout)

	env.out.Println("Merging data ...")

	for result := range env.backend.MergeAll(remote) {
		if result.Err != nil {
			env.err.Println(result.Err)
		}

		if result.Status != entity.MergeStatusNothing {
			env.out.Printf("%s: %s\n", result.Id.Human(), result)
		}
	}

	return nil
}
