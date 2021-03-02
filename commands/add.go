package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/input"

	"time"
)

type addOptions struct {
	title       string
	message     string
	messageFile string
	unixTime    int64
}

func newAddCommand() *cobra.Command {
	env := newEnv()
	options := addOptions{}

	cmd := &cobra.Command{
		Use:      "add",
		Short:    "Create a new bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(env, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue")
	flags.StringVarP(&options.message, "message", "m", "",
		"Provide a message to describe the issue")
	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")
	flags.Int64VarP(&options.unixTime, "time", "u", 0,
		"Set the unix timestamp of the commit, in seconds since 1970-01-01")

	return cmd
}

func runAdd(env *Env, opts addOptions) error {
	var err error
	if opts.messageFile != "" && opts.message == "" {
		opts.title, opts.message, err = input.BugCreateFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if opts.messageFile == "" && (opts.message == "" || opts.title == "") {
		opts.title, opts.message, err = input.BugCreateEditorInput(env.backend, opts.title, opts.message)

		if err == input.ErrEmptyTitle {
			env.out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	if opts.unixTime == 0 {
		opts.unixTime = time.Now().Unix()
	}

	b, _, err := env.backend.NewBugWithFilesAndTime(opts.unixTime, opts.title, opts.message, nil)

	if err != nil {
		return err
	}

	env.out.Printf("%s created\n", b.Id().Human())

	return nil
}
