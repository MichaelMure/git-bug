package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/input"
)

type commentEditOptions struct {
	messageFile    string
	message        string
	nonInteractive bool
}

func newCommentEditCommand() *cobra.Command {
	env := newEnv()
	options := commentEditOptions{}

	cmd := &cobra.Command{
		Use:     "edit [COMMENT_ID]",
		Short:   "Edit an existing comment on a bug.",
		Args:    cobra.ExactArgs(1),
		PreRunE: loadBackendEnsureUser(env),
		RunE: closeBackend(env, func(cmd *cobra.Command, args []string) error {
			return runCommentEdit(env, options, args)
		}),
		ValidArgsFunction: completeComment(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")

	flags.StringVarP(&options.message, "message", "m", "",
		"Provide the new message from the command line")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runCommentEdit(env *Env, opts commentEditOptions, args []string) error {
	b, commentId, err := env.backend.ResolveComment(args[0])
	if err != nil {
		return err
	}

	if opts.messageFile != "" && opts.message == "" {
		opts.message, err = input.BugCommentFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if opts.messageFile == "" && opts.message == "" {
		if opts.nonInteractive {
			env.err.Println("No message given. Use -m or -F option to specify a message. Aborting.")
			return nil
		}
		opts.message, err = input.BugCommentEditorInput(env.backend, "")
		if err == input.ErrEmptyMessage {
			env.err.Println("Empty message, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	_, err = b.EditComment(commentId, opts.message)
	if err != nil {
		return err
	}

	return b.Commit()
}
