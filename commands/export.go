package commands

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
)

type exportOptions struct {
	queryOptions
}

func newExportCommand() *cobra.Command {
	env := newEnv()
	options := exportOptions{}

	cmd := &cobra.Command{
		Use:   "export [QUERY]",
		Short: "Export bugs as a series of operations.",
		Long: `Export bugs as a series of operations.

The output format is NDJSON (Newline delimited JSON).

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language, flags, a natural language full text search, or a combination of the aforementioned.`,
		Example:  `See ls`,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringSliceVarP(&options.statusQuery, "status", "s", nil,
		"Filter by status. Valid values are [open,closed]")
	flags.StringSliceVarP(&options.authorQuery, "author", "a", nil,
		"Filter by author")
	flags.StringSliceVarP(&options.participantQuery, "participant", "p", nil,
		"Filter by participant")
	flags.StringSliceVarP(&options.actorQuery, "actor", "A", nil,
		"Filter by actor")
	flags.StringSliceVarP(&options.labelQuery, "label", "l", nil,
		"Filter by label")
	flags.StringSliceVarP(&options.titleQuery, "title", "t", nil,
		"Filter by title")
	flags.StringSliceVarP(&options.noQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	flags.StringVarP(&options.sortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	flags.StringVarP(&options.sortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")

	return cmd
}

func runExport(env *Env, opts exportOptions, args []string) error {
	q, err := makeQuery(args, &opts.queryOptions)
	if err != nil {
		return err
	}

	allIds := env.backend.QueryBugs(q)

	out := json.NewEncoder(env.out)

	for _, id := range allIds {
		b, err := env.backend.ResolveBug(id)
		if err != nil {
			return err
		}

		wrapper := struct {
			Id         entity.Id       `json:"id"`
			Operations []bug.Operation `json:"operations"`
		}{
			Id:         id,
			Operations: b.Snapshot().Operations,
		}

		err = out.Encode(wrapper)
		if err != nil {
			return err
		}
	}

	return nil
}
