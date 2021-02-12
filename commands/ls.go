package commands

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/util/colors"
)

type lsOptions struct {
	statusQuery      []string
	authorQuery      []string
	participantQuery []string
	actorQuery       []string
	labelQuery       []string
	titleQuery       []string
	noQuery          []string
	sortBy           string
	sortDirection    string
	outputFormat     string
}

func newLsCommand() *cobra.Command {
	env := newEnv()
	options := lsOptions{}

	cmd := &cobra.Command{
		Use:   "ls [QUERY]",
		Short: "List bugs.",
		Long: `Display a summary of each bugs.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language, flags, a natural language full text search, or a combination of the aforementioned.`,
		Example: `List open bugs sorted by last edition with a query:
git bug ls status:open sort:edit-desc

List closed bugs sorted by creation with flags:
git bug ls --status closed --by creation

Do a full text search of all bugs:
git bug ls "foo bar" baz

Use queries, flags, and full text search:
git bug ls status:open --by creation "foo bar" baz
`,
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLs(env, options, args)
		},
		ValidArgsFunction: completeLs(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringSliceVarP(&options.statusQuery, "status", "s", nil,
		"Filter by status. Valid values are [open,closed]")
	cmd.RegisterFlagCompletionFunc("status", completeFrom([]string{"open", "closed"}))
	flags.StringSliceVarP(&options.authorQuery, "author", "a", nil,
		"Filter by author")
	cmd.RegisterFlagCompletionFunc("author", completeUserForQuery(env))
	flags.StringSliceVarP(&options.participantQuery, "participant", "p", nil,
		"Filter by participant")
	cmd.RegisterFlagCompletionFunc("participant", completeUserForQuery(env))
	flags.StringSliceVarP(&options.actorQuery, "actor", "A", nil,
		"Filter by actor")
	cmd.RegisterFlagCompletionFunc("actor", completeUserForQuery(env))
	flags.StringSliceVarP(&options.labelQuery, "label", "l", nil,
		"Filter by label")
	cmd.RegisterFlagCompletionFunc("label", completeLabel(env))
	flags.StringSliceVarP(&options.titleQuery, "title", "t", nil,
		"Filter by title")
	flags.StringSliceVarP(&options.noQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	cmd.RegisterFlagCompletionFunc("no", completeLabel(env))
	flags.StringVarP(&options.sortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	cmd.RegisterFlagCompletionFunc("by", completeFrom([]string{"id", "creation", "edit"}))
	flags.StringVarP(&options.sortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")
	cmd.RegisterFlagCompletionFunc("direction", completeFrom([]string{"asc", "desc"}))
	flags.StringVarP(&options.outputFormat, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,plain,json,org-mode]")
	cmd.RegisterFlagCompletionFunc("format",
		completeFrom([]string{"default", "plain", "json", "org-mode"}))

	return cmd
}

func runLs(env *Env, opts lsOptions, args []string) error {
	var q *query.Query
	var err error

	if len(args) >= 1 {
		// either the shell or cobra remove the quotes, we need them back for the parsing
		for i, arg := range args {
			if strings.Contains(arg, " ") {
				args[i] = fmt.Sprintf("\"%s\"", arg)
			}
		}
		assembled := strings.Join(args, " ")
		q, err = query.Parse(assembled)
		if err != nil {
			return err
		}
	} else {
		q = query.NewQuery()
	}

	err = completeQuery(q, opts)
	if err != nil {
		return err
	}

	allIds, err := env.backend.QueryBugs(q)
	if err != nil {
		return err
	}

	bugExcerpt := make([]*cache.BugExcerpt, len(allIds))
	for i, id := range allIds {
		b, err := env.backend.ResolveBugExcerpt(id)
		if err != nil {
			return err
		}
		bugExcerpt[i] = b
	}

	switch opts.outputFormat {
	case "org-mode":
		return lsOrgmodeFormatter(env, bugExcerpt)
	case "plain":
		return lsPlainFormatter(env, bugExcerpt)
	case "json":
		return lsJsonFormatter(env, bugExcerpt)
	case "default":
		return lsDefaultFormatter(env, bugExcerpt)
	default:
		return fmt.Errorf("unknown format %s", opts.outputFormat)
	}
}

type JSONBugExcerpt struct {
	Id         string   `json:"id"`
	HumanId    string   `json:"human_id"`
	CreateTime JSONTime `json:"create_time"`
	EditTime   JSONTime `json:"edit_time"`

	Status       string         `json:"status"`
	Labels       []bug.Label    `json:"labels"`
	Title        string         `json:"title"`
	Actors       []JSONIdentity `json:"actors"`
	Participants []JSONIdentity `json:"participants"`
	Author       JSONIdentity   `json:"author"`

	Comments int               `json:"comments"`
	Metadata map[string]string `json:"metadata"`
}

func lsJsonFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	jsonBugs := make([]JSONBugExcerpt, len(bugExcerpts))
	for i, b := range bugExcerpts {
		jsonBug := JSONBugExcerpt{
			Id:         b.Id.String(),
			HumanId:    b.Id.Human(),
			CreateTime: NewJSONTime(b.CreateTime(), b.CreateLamportTime),
			EditTime:   NewJSONTime(b.EditTime(), b.EditLamportTime),
			Status:     b.Status.String(),
			Labels:     b.Labels,
			Title:      b.Title,
			Comments:   b.LenComments,
			Metadata:   b.CreateMetadata,
		}

		author, err := env.backend.ResolveIdentityExcerpt(b.AuthorId)
		if err != nil {
			return err
		}
		jsonBug.Author = NewJSONIdentityFromExcerpt(author)

		jsonBug.Actors = make([]JSONIdentity, len(b.Actors))
		for i, element := range b.Actors {
			actor, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}
			jsonBug.Actors[i] = NewJSONIdentityFromExcerpt(actor)
		}

		jsonBug.Participants = make([]JSONIdentity, len(b.Participants))
		for i, element := range b.Participants {
			participant, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}
			jsonBug.Participants[i] = NewJSONIdentityFromExcerpt(participant)
		}

		jsonBugs[i] = jsonBug
	}
	jsonObject, _ := json.MarshalIndent(jsonBugs, "", "    ")
	env.out.Printf("%s\n", jsonObject)
	return nil
}

func lsDefaultFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	for _, b := range bugExcerpts {
		author, err := env.backend.ResolveIdentityExcerpt(b.AuthorId)
		if err != nil {
			return err
		}

		var labelsTxt strings.Builder
		for _, l := range b.Labels {
			lc256 := l.Color().Term256()
			labelsTxt.WriteString(lc256.Escape())
			labelsTxt.WriteString(" ◼")
			labelsTxt.WriteString(lc256.Unescape())
		}

		// truncate + pad if needed
		labelsFmt := text.TruncateMax(labelsTxt.String(), 10)
		titleFmt := text.LeftPadMaxLine(strings.TrimSpace(b.Title), 50-text.Len(labelsFmt), 0)
		authorFmt := text.LeftPadMaxLine(author.DisplayName(), 15, 0)

		comments := fmt.Sprintf("%3d 💬", b.LenComments-1)
		if b.LenComments-1 <= 0 {
			comments = ""
		}
		if b.LenComments-1 > 999 {
			comments = "  ∞ 💬"
		}

		env.out.Printf("%s %s\t%s\t%s\t%s\n",
			colors.Cyan(b.Id.Human()),
			colors.Yellow(b.Status),
			titleFmt+labelsFmt,
			colors.Magenta(authorFmt),
			comments,
		)
	}
	return nil
}

func lsPlainFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	for _, b := range bugExcerpts {
		env.out.Printf("%s [%s] %s\n", b.Id.Human(), b.Status, strings.TrimSpace(b.Title))
	}
	return nil
}

func lsOrgmodeFormatter(env *Env, bugExcerpts []*cache.BugExcerpt) error {
	// see https://orgmode.org/manual/Tags.html
	orgTagRe := regexp.MustCompile("[^[:alpha:]_@]")
	formatTag := func(l bug.Label) string {
		return orgTagRe.ReplaceAllString(l.String(), "_")
	}

	formatTime := func(time time.Time) string {
		return time.Format("[2006-01-02 Mon 15:05]")
	}

	env.out.Println("#+TODO: OPEN | CLOSED")

	for _, b := range bugExcerpts {
		status := strings.ToUpper(b.Status.String())

		var title string
		if link, ok := b.CreateMetadata["github-url"]; ok {
			title = fmt.Sprintf("[[%s][%s]]", link, b.Title)
		} else {
			title = b.Title
		}

		author, err := env.backend.ResolveIdentityExcerpt(b.AuthorId)
		if err != nil {
			return err
		}

		var labels strings.Builder
		labels.WriteString(":")
		for i, l := range b.Labels {
			if i > 0 {
				labels.WriteString(":")
			}
			labels.WriteString(formatTag(l))
		}
		labels.WriteString(":")

		env.out.Printf("* %-6s %s %s %s: %s %s\n",
			status,
			b.Id.Human(),
			formatTime(b.CreateTime()),
			author.DisplayName(),
			title,
			labels.String(),
		)

		env.out.Printf("** Last Edited: %s\n", formatTime(b.EditTime()))

		env.out.Printf("** Actors:\n")
		for _, element := range b.Actors {
			actor, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}

			env.out.Printf(": %s %s\n",
				actor.Id.Human(),
				actor.DisplayName(),
			)
		}

		env.out.Printf("** Participants:\n")
		for _, element := range b.Participants {
			participant, err := env.backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}

			env.out.Printf(": %s %s\n",
				participant.Id.Human(),
				participant.DisplayName(),
			)
		}
	}

	return nil
}

// Finish the command flags transformation into the query.Query
func completeQuery(q *query.Query, opts lsOptions) error {
	for _, str := range opts.statusQuery {
		status, err := bug.StatusFromString(str)
		if err != nil {
			return err
		}
		q.Status = append(q.Status, status)
	}

	q.Author = append(q.Author, opts.authorQuery...)
	q.Participant = append(q.Participant, opts.participantQuery...)
	q.Actor = append(q.Actor, opts.actorQuery...)
	q.Label = append(q.Label, opts.labelQuery...)
	q.Title = append(q.Title, opts.titleQuery...)

	for _, no := range opts.noQuery {
		switch no {
		case "label":
			q.NoLabel = true
		default:
			return fmt.Errorf("unknown \"no\" filter %s", no)
		}
	}

	switch opts.sortBy {
	case "id":
		q.OrderBy = query.OrderById
	case "creation":
		q.OrderBy = query.OrderByCreation
	case "edit":
		q.OrderBy = query.OrderByEdit
	default:
		return fmt.Errorf("unknown sort flag %s", opts.sortBy)
	}

	switch opts.sortDirection {
	case "asc":
		q.OrderDirection = query.OrderAscending
	case "desc":
		q.OrderDirection = query.OrderDescending
	default:
		return fmt.Errorf("unknown sort direction %s", opts.sortDirection)
	}

	return nil
}
