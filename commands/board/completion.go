package boardcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	bugcmd "github.com/MichaelMure/git-bug/commands/bug"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	_select "github.com/MichaelMure/git-bug/commands/select"
)

// BoardCompletion complete a board id
func BoardCompletion(env *execenv.Env) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		return boardWithBackend(env.Backend, toComplete)
	}
}

func boardWithBackend(backend *cache.RepoCache, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
	for _, id := range backend.Boards().AllIds() {
		if strings.Contains(id.String(), strings.TrimSpace(toComplete)) {
			excerpt, err := backend.Boards().ResolveExcerpt(id)
			if err != nil {
				return completion.HandleError(err)
			}
			completions = append(completions, id.Human()+"\t"+excerpt.Title)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// ColumnCompletion complete a board's column id
func ColumnCompletion(env *execenv.Env) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		b, _, err := ResolveSelected(env.Backend, args)
		switch {
		case _select.IsErrNoValidId(err):
			// no completion
		case err == nil:
			for _, column := range b.Snapshot().Columns {
				completions = append(completions, column.CombinedId.Human()+"\t"+column.Name)
			}
		default:
			return completion.HandleError(err)
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func BoardAndBugCompletion(env *execenv.Env) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		_, _, err := ResolveSelected(env.Backend, args)
		switch {
		case _select.IsErrNoValidId(err):
			return boardWithBackend(env.Backend, toComplete)
		case err == nil:
			return bugcmd.BugWithBackend(env.Backend, toComplete)
		default:
			return completion.HandleError(err)
		}

	}
}
