package boardcmd

import (
	"fmt"
	"strings"

	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/cmdjson"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/util/colors"
)

type boardOptions struct {
	metadataQuery []string
	actorQuery    []string
	titleQuery    []string
	outputFormat  string
}

func NewBoardCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := boardOptions{}

	cmd := &cobra.Command{
		Use:     "board",
		Short:   "List boards",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoard(env, options, args)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringSliceVarP(&options.metadataQuery, "metadata", "m", nil,
		"Filter by metadata. Example: github-url=URL")
	cmd.RegisterFlagCompletionFunc("author", completion.UserForQuery(env))
	flags.StringSliceVarP(&options.actorQuery, "actor", "A", nil,
		"Filter by actor")
	cmd.RegisterFlagCompletionFunc("actor", completion.UserForQuery(env))
	flags.StringSliceVarP(&options.titleQuery, "title", "t", nil,
		"Filter by title")
	flags.StringVarP(&options.outputFormat, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,plain,compact,id,json,org-mode]")
	cmd.RegisterFlagCompletionFunc("format",
		completion.From([]string{"default", "id", "json"}))

	const selectGroup = "select"
	cmd.AddGroup(&cobra.Group{ID: selectGroup, Title: "Implicit selection"})

	addCmdWithGroup := func(child *cobra.Command, groupID string) {
		cmd.AddCommand(child)
		child.GroupID = groupID
	}

	addCmdWithGroup(newBoardDeselectCommand(), selectGroup)
	addCmdWithGroup(newBoardSelectCommand(), selectGroup)

	cmd.AddCommand(newBoardNewCommand())
	cmd.AddCommand(newBoardRmCommand())
	cmd.AddCommand(newBoardShowCommand())
	cmd.AddCommand(newBoardDescriptionCommand())
	cmd.AddCommand(newBoardTitleCommand())
	cmd.AddCommand(newBoardAddDraftCommand())
	cmd.AddCommand(newBoardAddBugCommand())

	return cmd
}

func runBoard(env *execenv.Env, opts boardOptions, args []string) error {
	// TODO: query

	allIds := env.Backend.Boards().AllIds()

	excerpts := make([]*cache.BoardExcerpt, len(allIds))
	for i, id := range allIds {
		b, err := env.Backend.Boards().ResolveExcerpt(id)
		if err != nil {
			return err
		}
		excerpts[i] = b
	}

	switch opts.outputFormat {
	case "json":
		return boardJsonFormatter(env, excerpts)
	case "id":
		return boardIDFormatter(env, excerpts)
	case "default":
		return boardDefaultFormatter(env, excerpts)
	default:
		return fmt.Errorf("unknown format %s", opts.outputFormat)
	}
}

func boardIDFormatter(env *execenv.Env, excerpts []*cache.BoardExcerpt) error {
	for _, b := range excerpts {
		env.Out.Println(b.Id().String())
	}

	return nil
}

func boardDefaultFormatter(env *execenv.Env, excerpts []*cache.BoardExcerpt) error {
	for _, b := range excerpts {
		// truncate + pad if needed
		titleFmt := text.LeftPadMaxLine(strings.TrimSpace(b.Title), 50, 0)
		descFmt := text.LeftPadMaxLine(strings.TrimSpace(b.Description), 50, 0)

		var itemFmt string
		switch {
		case b.ItemCount < 1:
			itemFmt = "empty"
		case b.ItemCount < 1000:
			itemFmt = fmt.Sprintf("%3d 📝", b.ItemCount)
		default:
			itemFmt = "  ∞ 📝"

		}

		env.Out.Printf("%s\t%s\t%s\t%s\n",
			colors.Cyan(b.Id().Human()),
			titleFmt,
			descFmt,
			itemFmt,
		)
	}
	return nil
}

func boardJsonFormatter(env *execenv.Env, excerpts []*cache.BoardExcerpt) error {
	res := make([]cmdjson.BoardExcerpt, len(excerpts))
	for i, b := range excerpts {
		jsonBoard, err := cmdjson.NewBoardExcerpt(env.Backend, b)
		if err != nil {
			return err
		}
		res[i] = jsonBoard
	}
	return env.Out.PrintJSON(res)
}
