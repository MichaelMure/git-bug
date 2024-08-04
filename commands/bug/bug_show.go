package bugcmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/cmdjson"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/util/colors"
)

type bugShowOptions struct {
	fields string
	format string
}

func newBugShowCommand(env *execenv.Env) *cobra.Command {
	options := bugShowOptions{}

	cmd := &cobra.Command{
		Use:     "show [BUG_ID]",
		Short:   "Display the details of a bug",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugShow(env, options, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	fields := []string{"author", "authorEmail", "createTime", "lastEdit", "humanId",
		"id", "labels", "shortId", "status", "title", "actors", "participants"}
	flags.StringVarP(&options.fields, "field", "", "",
		"Select field to display. Valid values are ["+strings.Join(fields, ",")+"]")
	cmd.RegisterFlagCompletionFunc("by", completion.From(fields))
	flags.StringVarP(&options.format, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,json,org-mode]")

	return cmd
}

func runBugShow(env *execenv.Env, opts bugShowOptions, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Compile()

	if len(snap.Comments) == 0 {
		return errors.New("invalid bug: no comment")
	}

	if opts.fields != "" {
		switch opts.fields {
		case "author":
			env.Out.Printf("%s\n", snap.Author.DisplayName())
		case "authorEmail":
			env.Out.Printf("%s\n", snap.Author.Email())
		case "createTime":
			env.Out.Printf("%s\n", snap.CreateTime.String())
		case "lastEdit":
			env.Out.Printf("%s\n", snap.EditTime().String())
		case "humanId":
			env.Out.Printf("%s\n", snap.Id().Human())
		case "id":
			env.Out.Printf("%s\n", snap.Id())
		case "labels":
			for _, l := range snap.Labels {
				env.Out.Printf("%s\n", l.String())
			}
		case "actors":
			for _, a := range snap.Actors {
				env.Out.Printf("%s\n", a.DisplayName())
			}
		case "participants":
			for _, p := range snap.Participants {
				env.Out.Printf("%s\n", p.DisplayName())
			}
		case "shortId":
			env.Out.Printf("%s\n", snap.Id().Human())
		case "status":
			env.Out.Printf("%s\n", snap.Status)
		case "title":
			env.Out.Printf("%s\n", snap.Title)
		default:
			return fmt.Errorf("unsupported field: %s", opts.fields)
		}

		return nil
	}

	switch opts.format {
	case "org-mode":
		return showOrgModeFormatter(env, snap)
	case "json":
		return showJsonFormatter(env, snap)
	case "default":
		return showDefaultFormatter(env, snap)
	default:
		return fmt.Errorf("unknown format %s", opts.format)
	}
}

func showDefaultFormatter(env *execenv.Env, snapshot *bug.Snapshot) error {
	// Header
	env.Out.Printf("%s [%s] %s\n\n",
		colors.Cyan(snapshot.Id().Human()),
		colors.Yellow(snapshot.Status),
		snapshot.Title,
	)

	env.Out.Printf("%s opened this issue %s\n",
		colors.Magenta(snapshot.Author.DisplayName()),
		snapshot.CreateTime.String(),
	)

	env.Out.Printf("This was last edited at %s\n\n",
		snapshot.EditTime().String(),
	)

	// Labels
	var labels = make([]string, len(snapshot.Labels))
	for i := range snapshot.Labels {
		labels[i] = string(snapshot.Labels[i])
	}

	env.Out.Printf("labels: %s\n",
		strings.Join(labels, ", "),
	)

	// Actors
	var actors = make([]string, len(snapshot.Actors))
	for i := range snapshot.Actors {
		actors[i] = snapshot.Actors[i].DisplayName()
	}

	env.Out.Printf("actors: %s\n",
		strings.Join(actors, ", "),
	)

	// Participants
	var participants = make([]string, len(snapshot.Participants))
	for i := range snapshot.Participants {
		participants[i] = snapshot.Participants[i].DisplayName()
	}

	env.Out.Printf("participants: %s\n\n",
		strings.Join(participants, ", "),
	)

	// Comments
	indent := "  "

	for i, comment := range snapshot.Comments {
		var message string
		env.Out.Printf("%s%s #%d %s <%s>\n\n",
			indent,
			comment.CombinedId().Human(),
			i,
			comment.Author.DisplayName(),
			comment.Author.Email(),
		)

		if comment.Message == "" {
			message = colors.BlackBold(colors.WhiteBg("No description provided."))
		} else {
			message = comment.Message
		}

		env.Out.Printf("%s%s\n\n\n",
			indent,
			message,
		)
	}

	return nil
}

func showJsonFormatter(env *execenv.Env, snap *bug.Snapshot) error {
	jsonBug := cmdjson.NewBugSnapshot(snap)
	return env.Out.PrintJSON(jsonBug)
}

func showOrgModeFormatter(env *execenv.Env, snapshot *bug.Snapshot) error {
	// Header
	env.Out.Printf("%s [%s] %s\n",
		snapshot.Id().Human(),
		snapshot.Status,
		snapshot.Title,
	)

	env.Out.Printf("* Author: %s\n",
		snapshot.Author.DisplayName(),
	)

	env.Out.Printf("* Creation Time: %s\n",
		snapshot.CreateTime.String(),
	)

	env.Out.Printf("* Last Edit: %s\n",
		snapshot.EditTime().String(),
	)

	// Labels
	var labels = make([]string, len(snapshot.Labels))
	for i, label := range snapshot.Labels {
		labels[i] = string(label)
	}

	env.Out.Printf("* Labels:\n")
	if len(labels) > 0 {
		env.Out.Printf("** %s\n",
			strings.Join(labels, "\n** "),
		)
	}

	// Actors
	var actors = make([]string, len(snapshot.Actors))
	for i, actor := range snapshot.Actors {
		actors[i] = fmt.Sprintf("%s %s",
			actor.Id().Human(),
			actor.DisplayName(),
		)
	}

	env.Out.Printf("* Actors:\n** %s\n",
		strings.Join(actors, "\n** "),
	)

	// Participants
	var participants = make([]string, len(snapshot.Participants))
	for i, participant := range snapshot.Participants {
		participants[i] = fmt.Sprintf("%s %s",
			participant.Id().Human(),
			participant.DisplayName(),
		)
	}

	env.Out.Printf("* Participants:\n** %s\n",
		strings.Join(participants, "\n** "),
	)

	env.Out.Printf("* Comments:\n")

	for i, comment := range snapshot.Comments {
		var message string
		env.Out.Printf("** #%d %s\n",
			i, comment.Author.DisplayName())

		if comment.Message == "" {
			message = "No description provided."
		} else {
			message = strings.ReplaceAll(comment.Message, "\n", "\n: ")
		}

		env.Out.Printf(": %s\n", message)
	}

	return nil
}
