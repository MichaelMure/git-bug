package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
)

type userOptions struct {
	fields string
}

func newUserCommand() *cobra.Command {
	env := newEnv()
	options := userOptions{}

	cmd := &cobra.Command{
		Use:      "user [USER-ID]",
		Short:    "Display or change the user identity.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUser(env, options, args)
		},
		ValidArgsFunction: completeUser(env),
	}

	cmd.AddCommand(newUserAdoptCommand())
	cmd.AddCommand(newUserCreateCommand())
	cmd.AddCommand(newUserLsCommand())

	flags := cmd.Flags()
	flags.SortFlags = false

	fields := []string{"email", "humanId", "id", "lastModification", "lastModificationLamport", "login", "metadata", "name"}
	flags.StringVarP(&options.fields, "field", "f", "",
		"Select field to display. Valid values are ["+strings.Join(fields, ",")+"]")
	cmd.RegisterFlagCompletionFunc("field", completeFrom(fields))

	return cmd
}

func runUser(env *Env, opts userOptions, args []string) error {
	if len(args) > 1 {
		return errors.New("only one identity can be displayed at a time")
	}

	var id *cache.IdentityCache
	var err error
	if len(args) == 1 {
		id, err = env.backend.ResolveIdentityPrefix(args[0])
	} else {
		id, err = env.backend.GetUserIdentity()
	}

	if err != nil {
		return err
	}

	if opts.fields != "" {
		switch opts.fields {
		case "email":
			env.out.Printf("%s\n", id.Email())
		case "login":
			env.out.Printf("%s\n", id.Login())
		case "humanId":
			env.out.Printf("%s\n", id.Id().Human())
		case "id":
			env.out.Printf("%s\n", id.Id())
		case "lastModification":
			env.out.Printf("%s\n", id.LastModification().
				Time().Format("Mon Jan 2 15:04:05 2006 +0200"))
		case "lastModificationLamport":
			env.out.Printf("%d\n", id.LastModificationLamport())
		case "metadata":
			for key, value := range id.ImmutableMetadata() {
				env.out.Printf("%s\n%s\n", key, value)
			}
		case "name":
			env.out.Printf("%s\n", id.Name())

		default:
			return fmt.Errorf("\nUnsupported field: %s\n", opts.fields)
		}

		return nil
	}

	env.out.Printf("Id: %s\n", id.Id())
	env.out.Printf("Name: %s\n", id.Name())
	env.out.Printf("Email: %s\n", id.Email())
	env.out.Printf("Login: %s\n", id.Login())
	env.out.Printf("Last modification: %s (lamport %d)\n",
		id.LastModification().Time().Format("Mon Jan 2 15:04:05 2006 +0200"),
		id.LastModificationLamport())
	env.out.Println("Metadata:")
	for key, value := range id.ImmutableMetadata() {
		env.out.Printf("    %s --> %s\n", key, value)
	}
	// env.out.Printf("Protected: %v\n", id.IsProtected())

	return nil
}
