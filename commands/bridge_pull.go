package commands

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/araddon/dateparse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

type bridgePullOptions struct {
	importSince string
	noResume    bool
}

func newBridgePullCommand() *cobra.Command {
	env := newEnv()
	options := bridgePullOptions{}

	cmd := &cobra.Command{
		Use:      "pull [NAME]",
		Short:    "Pull updates.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgePull(env, options, args)
		},
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeBridge(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&options.noResume, "no-resume", "n", false, "force importing all bugs")
	flags.StringVarP(&options.importSince, "since", "s", "", "import only bugs updated after the given date (ex: \"200h\" or \"june 2 2019\")")

	return cmd
}

func runBridgePull(env *Env, opts bridgePullOptions, args []string) error {
	if opts.noResume && opts.importSince != "" {
		return fmt.Errorf("only one of --no-resume and --since flags should be used")
	}

	var b *core.Bridge
	var err error

	if len(args) == 0 {
		b, err = bridge.DefaultBridge(env.backend)
	} else {
		b, err = bridge.LoadBridge(env.backend, args[0])
	}

	if err != nil {
		return err
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// buffered channel to avoid send block at the end
	done := make(chan struct{}, 1)

	var mu sync.Mutex
	interruptCount := 0
	interrupt.RegisterCleaner(func() error {
		mu.Lock()
		if interruptCount > 0 {
			env.err.Println("Received another interrupt before graceful stop, terminating...")
			os.Exit(0)
		}

		interruptCount++
		mu.Unlock()

		env.err.Println("Received interrupt signal, stopping the import...\n(Hit ctrl-c again to kill the process.)")

		// send signal to stop the importer
		cancel()

		// block until importer gracefully shutdown
		<-done
		return nil
	})

	var events <-chan core.ImportResult
	switch {
	case opts.noResume:
		events, err = b.ImportAllSince(ctx, time.Time{})
	case opts.importSince != "":
		since, err2 := parseSince(opts.importSince)
		if err2 != nil {
			return errors.Wrap(err2, "import time parsing")
		}
		events, err = b.ImportAllSince(ctx, since)
	default:
		events, err = b.ImportAll(ctx)
	}

	if err != nil {
		return err
	}

	importedIssues := 0
	importedIdentities := 0
	for result := range events {
		switch result.Event {
		case core.ImportEventNothing:
			// filtered

		case core.ImportEventBug:
			importedIssues++
			env.out.Println(result.String())

		case core.ImportEventIdentity:
			importedIdentities++
			env.out.Println(result.String())

		case core.ImportEventError:
			if result.Err != context.Canceled {
				env.out.Println(result.String())
			}

		default:
			env.out.Println(result.String())
		}
	}

	env.out.Printf("imported %d issues and %d identities with %s bridge\n", importedIssues, importedIdentities, b.Name)

	// send done signal
	close(done)

	return nil
}

func parseSince(since string) (time.Time, error) {
	duration, err := time.ParseDuration(since)
	if err == nil {
		return time.Now().Add(-duration), nil
	}

	return dateparse.ParseLocal(since)
}
