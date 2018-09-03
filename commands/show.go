package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util"
	"github.com/spf13/cobra"
)

func runShowBug(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("Only showing one bug at a time is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	prefix := args[0]

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	snapshot := b.Snapshot()

	if len(snapshot.Comments) == 0 {
		return errors.New("Invalid bug: no comment")
	}

	firstComment := snapshot.Comments[0]

	// Header
	fmt.Printf("[%s] %s %s\n\n",
		util.Yellow(snapshot.Status),
		util.Cyan(snapshot.HumanId()),
		snapshot.Title,
	)

	fmt.Printf("%s opened this issue %s\n\n",
		util.Magenta(firstComment.Author.Name),
		firstComment.FormatTime(),
	)

	var labels = make([]string, len(snapshot.Labels))
	for i := range snapshot.Labels {
		labels[i] = string(snapshot.Labels[i])
	}

	fmt.Printf("labels: %s\n\n",
		strings.Join(labels, ", "),
	)

	// Comments
	indent := "  "

	for i, comment := range snapshot.Comments {
		fmt.Printf("%s#%d %s <%s>\n\n",
			indent,
			i,
			comment.Author.Name,
			comment.Author.Email,
		)

		fmt.Printf("%s%s\n\n\n",
			indent,
			comment.Message,
		)
	}

	return nil
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display the details of a bug",
	RunE:  runShowBug,
}

func init() {
	RootCmd.AddCommand(showCmd)
}
