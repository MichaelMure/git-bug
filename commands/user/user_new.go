package usercmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/commands/input"
)

type userNewOptions struct {
	name           string
	email          string
	avatarURL      string
	nonInteractive bool
}

func newUserNewCommand(env *execenv.Env) *cobra.Command {
	options := userNewOptions{}
	cmd := &cobra.Command{
		Use:     "new",
		Short:   "Create a new identity",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runUserNew(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.name, "name", "n", "", "Name to identify the user")
	flags.StringVarP(&options.email, "email", "e", "", "Email of the user")
	flags.StringVarP(&options.avatarURL, "avatar", "a", "", "Avatar URL")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runUserNew(env *execenv.Env, opts userNewOptions) error {

	if !opts.nonInteractive && opts.name == "" {
		preName, err := env.Backend.GetUserName()
		if err != nil {
			return err
		}
		opts.name, err = input.PromptDefault("Name", "name", preName, input.Required)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.email == "" {
		preEmail, err := env.Backend.GetUserEmail()
		if err != nil {
			return err
		}

		opts.email, err = input.PromptDefault("Email", "email", preEmail, input.Required)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.avatarURL == "" {
		var err error
		opts.avatarURL, err = input.Prompt("Avatar URL", "avatar")
		if err != nil {
			return err
		}
	}

	id, err := env.Backend.Identities().NewRaw(opts.name, opts.email, "", opts.avatarURL, nil, nil)
	if err != nil {
		return err
	}

	err = id.CommitAsNeeded()
	if err != nil {
		return err
	}

	set, err := env.Backend.IsUserIdentitySet()
	if err != nil {
		return err
	}

	if !set {
		err = env.Backend.SetUserIdentity(id)
		if err != nil {
			return err
		}
	}

	env.Err.Println()
	env.Out.Print(id.Id())
	env.Err.Println()

	return nil
}
