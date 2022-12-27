package boardcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
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

		for _, id := range env.Backend.Boards().AllIds() {
			if strings.Contains(id.String(), strings.TrimSpace(toComplete)) {
				excerpt, err := env.Backend.Boards().ResolveExcerpt(id)
				if err != nil {
					return completion.HandleError(err)
				}
				completions = append(completions, id.Human()+"\t"+excerpt.Title)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func ColumnCompletion(env *execenv.Env) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		b, _, err := ResolveSelected(env.Backend, args)
		if err != nil {
			return completion.HandleError(err)
		}

		for _, column := range b.Snapshot().Columns {
			completions = append(completions, column.Id.Human()+"\t"+column.Name)
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
