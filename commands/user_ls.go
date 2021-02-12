package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
)

type userLsOptions struct {
	format string
}

func newUserLsCommand() *cobra.Command {
	env := newEnv()
	options := userLsOptions{}

	cmd := &cobra.Command{
		Use:      "ls",
		Short:    "List identities.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserLs(env, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.format, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,json]")
	cmd.RegisterFlagCompletionFunc("format", completeFrom([]string{"default", "json"}))

	return cmd
}

func runUserLs(env *Env, opts userLsOptions) error {
	ids := env.backend.AllIdentityIds()
	var users []*cache.IdentityExcerpt
	for _, id := range ids {
		user, err := env.backend.ResolveIdentityExcerpt(id)
		if err != nil {
			return err
		}
		users = append(users, user)
	}

	switch opts.format {
	case "json":
		return userLsJsonFormatter(env, users)
	case "default":
		return userLsDefaultFormatter(env, users)
	default:
		return fmt.Errorf("unknown format %s", opts.format)
	}
}

func userLsDefaultFormatter(env *Env, users []*cache.IdentityExcerpt) error {
	for _, user := range users {
		env.out.Printf("%s %s\n",
			colors.Cyan(user.Id.Human()),
			user.DisplayName(),
		)
	}

	return nil
}

func userLsJsonFormatter(env *Env, users []*cache.IdentityExcerpt) error {
	jsonUsers := make([]JSONIdentity, len(users))
	for i, user := range users {
		jsonUsers[i] = NewJSONIdentityFromExcerpt(user)
	}

	jsonObject, _ := json.MarshalIndent(jsonUsers, "", "    ")
	env.out.Printf("%s\n", jsonObject)
	return nil
}
