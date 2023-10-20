package bugcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entities/bug"
)

// BugCompletion complete a bug id
func BugCompletion(env *execenv.Env) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		return bugWithBackend(env.Backend, toComplete)
	}
}

func bugWithBackend(backend *cache.RepoCache, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
	for _, id := range backend.Bugs().AllIds() {
		if strings.Contains(id.String(), strings.TrimSpace(toComplete)) {
			excerpt, err := backend.Bugs().ResolveExcerpt(id)
			if err != nil {
				return completion.HandleError(err)
			}
			completions = append(completions, id.Human()+"\t"+excerpt.Title)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// BugAndLabelsCompletion complete either a bug ID or a label if we know about the bug
func BugAndLabelsCompletion(env *execenv.Env, addOrRemove bool) completion.ValidArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := execenv.LoadBackend(env)(cmd, args); err != nil {
			return completion.HandleError(err)
		}
		defer func() {
			_ = env.Backend.Close()
		}()

		b, cleanArgs, err := ResolveSelected(env.Backend, args)
		if _select.IsErrNoValidId(err) {
			// we need a bug first to complete labels
			return bugWithBackend(env.Backend, toComplete)
		}
		if err != nil {
			return completion.HandleError(err)
		}

		snap := b.Compile()

		seenLabels := map[bug.Label]bool{}
		for _, label := range cleanArgs {
			seenLabels[bug.Label(label)] = addOrRemove
		}

		var labels []bug.Label
		if addOrRemove {
			for _, label := range snap.Labels {
				seenLabels[label] = true
			}

			allLabels := env.Backend.Bugs().ValidLabels()
			labels = make([]bug.Label, 0, len(allLabels))
			for _, label := range allLabels {
				if !seenLabels[label] {
					labels = append(labels, label)
				}
			}
		} else {
			labels = make([]bug.Label, 0, len(snap.Labels))
			for _, label := range snap.Labels {
				if seenLabels[label] {
					labels = append(labels, label)
				}
			}
		}

		completions = make([]string, len(labels))
		for i, label := range labels {
			completions[i] = string(label) + "\t" + "Label"
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
