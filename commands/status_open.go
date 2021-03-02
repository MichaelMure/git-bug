package commands

import (
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"

	"time"
)

type statusOpenOptions struct {
	unixTime    int64
}

func newStatusOpenCommand() *cobra.Command {
	env := newEnv()
	options := statusOpenOptions{}

	cmd := &cobra.Command{
		Use:      "open [ID]",
		Short:    "Mark a bug as open.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusOpen(env, args, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.Int64VarP(&options.unixTime, "time", "u", 0,
		"Set the unix timestamp of a status change, in seconds since 1970-01-01")

	return cmd
}

func runStatusOpen(env *Env, args []string, opts statusOpenOptions) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	if opts.unixTime == 0 {
		opts.unixTime = time.Now().Unix()
	}

	_, err = b.OpenWithTime(opts.unixTime)
	if err != nil {
		return err
	}

	return b.Commit()
}
