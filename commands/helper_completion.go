package commands

import (
	"fmt"
	"sort"
	"strings"

	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entity"
)

type validArgsFunction func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective)

func completionHandlerError(err error) (completions []string, directives cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveError
}

func completeBridge(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		bridges, err := bridge.ConfiguredBridges(env.backend)
		if err != nil {
			return completionHandlerError(err)
		}

		completions = make([]string, len(bridges))
		for i, bridge := range bridges {
			completions[i] = bridge + "\t" + "Bridge"
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeBridgeAuth(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		creds, err := auth.List(env.backend)
		if err != nil {
			return completionHandlerError(err)
		}

		completions = make([]string, len(creds))
		for i, cred := range creds {
			meta := make([]string, 0, len(cred.Metadata()))
			for k, v := range cred.Metadata() {
				meta = append(meta, k+":"+v)
			}
			sort.Strings(meta)
			metaFmt := strings.Join(meta, ",")

			completions[i] = cred.ID().Human() + "\t" + cred.Target() + " " + string(cred.Kind()) + " " + metaFmt
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeBug(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		return completeBugWithBackend(env.backend, toComplete)
	}
}

func completeBugWithBackend(backend *cache.RepoCache, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
	allIds := backend.AllBugsIds()
	bugExcerpt := make([]*cache.BugExcerpt, len(allIds))
	for i, id := range allIds {
		var err error
		bugExcerpt[i], err = backend.ResolveBugExcerpt(id)
		if err != nil {
			return completionHandlerError(err)
		}
	}

	for i, id := range allIds {
		if strings.Contains(id.String(), strings.TrimSpace(toComplete)) {
			completions = append(completions, id.Human()+"\t"+bugExcerpt[i].Title)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeBugAndLabels(env *Env, addOrRemove bool) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		b, args, err := _select.ResolveBug(env.backend, args)
		if err == _select.ErrNoValidId {
			// we need a bug first to complete labels
			return completeBugWithBackend(env.backend, toComplete)
		}
		if err != nil {
			return completionHandlerError(err)
		}

		snap := b.Snapshot()

		seenLabels := map[bug.Label]bool{}
		for _, label := range args {
			seenLabels[bug.Label(label)] = addOrRemove
		}

		var labels []bug.Label
		if addOrRemove {
			for _, label := range snap.Labels {
				seenLabels[label] = true
			}

			allLabels := env.backend.ValidLabels()
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

func completeComment(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		bugPrefix, _ := entity.SeparateIds(toComplete)
		bugCandidate := make([]entity.Id, 0, 5)

		// build a list of possible matching bugs
		_, _ = env.backend.ResolveBugExcerptMatcher(func(excerpt *cache.BugExcerpt) bool {
			if excerpt.Id.HasPrefix(bugPrefix) {
				bugCandidate = append(bugCandidate, excerpt.Id)
			}
			return false
		})

		completions = make([]string, 0, 5)

		for _, bugId := range bugCandidate {
			b, err := env.backend.ResolveBug(bugId)
			if err != nil {
				return completionHandlerError(err)
			}

			for _, comment := range b.Snapshot().Comments {
				if comment.CombinedId().HasPrefix(toComplete) {
					text.TrimSpace()
					message := strings.ReplaceAll(comment.Message, "\n", "")

					completions = append(completions, comment.CombinedId().Human()+"\t"+text.TruncateMax())
				}
			}
		}

	}
}

func completeFrom(choices []string) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return choices, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeGitRemote(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		remoteMap, err := env.backend.GetRemotes()
		if err != nil {
			return completionHandlerError(err)
		}
		completions = make([]string, 0, len(remoteMap))
		for remote, url := range remoteMap {
			completions = append(completions, remote+"\t"+"Remote: "+url)
		}
		sort.Strings(completions)
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeLabel(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		labels := env.backend.ValidLabels()
		completions = make([]string, len(labels))
		for i, label := range labels {
			if strings.Contains(label.String(), " ") {
				completions[i] = fmt.Sprintf("\"%s\"\tLabel", label.String())
			} else {
				completions[i] = fmt.Sprintf("%s\tLabel", label.String())
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeLs(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if strings.HasPrefix(toComplete, "status:") {
			completions = append(completions, "status:open\tOpen bugs")
			completions = append(completions, "status:closed\tClosed bugs")
			return completions, cobra.ShellCompDirectiveDefault
		}

		byPerson := []string{"author:", "participant:", "actor:"}
		byLabel := []string{"label:", "no:"}
		needBackend := false
		for _, key := range append(byPerson, byLabel...) {
			if strings.HasPrefix(toComplete, key) {
				needBackend = true
			}
		}

		if needBackend {
			if err := loadBackend(env)(cmd, args); err != nil {
				return completionHandlerError(err)
			}
			defer func() {
				_ = env.backend.Close()
			}()
		}

		for _, key := range byPerson {
			if !strings.HasPrefix(toComplete, key) {
				continue
			}
			ids := env.backend.AllIdentityIds()
			completions = make([]string, len(ids))
			for i, id := range ids {
				user, err := env.backend.ResolveIdentityExcerpt(id)
				if err != nil {
					return completionHandlerError(err)
				}
				var handle string
				if user.Login != "" {
					handle = user.Login
				} else {
					// "author:John Doe" does not work yet, so use the first name.
					handle = strings.Split(user.Name, " ")[0]
				}
				completions[i] = key + handle + "\t" + user.DisplayName()
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		for _, key := range byLabel {
			if !strings.HasPrefix(toComplete, key) {
				continue
			}
			labels := env.backend.ValidLabels()
			completions = make([]string, len(labels))
			for i, label := range labels {
				if strings.Contains(label.String(), " ") {
					completions[i] = key + "\"" + string(label) + "\""
				} else {
					completions[i] = key + string(label)
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		completions = []string{
			"actor:\tFilter by actor",
			"author:\tFilter by author",
			"label:\tFilter by label",
			"no:\tExclude bugs by label",
			"participant:\tFilter by participant",
			"status:\tFilter by open/close status",
			"title:\tFilter by title",
		}
		return completions, cobra.ShellCompDirectiveNoSpace
	}
}

func completeUser(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		ids := env.backend.AllIdentityIds()
		completions = make([]string, len(ids))
		for i, id := range ids {
			user, err := env.backend.ResolveIdentityExcerpt(id)
			if err != nil {
				return completionHandlerError(err)
			}
			completions[i] = user.Id.Human() + "\t" + user.DisplayName()
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeUserForQuery(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = env.backend.Close()
		}()

		ids := env.backend.AllIdentityIds()
		completions = make([]string, len(ids))
		for i, id := range ids {
			user, err := env.backend.ResolveIdentityExcerpt(id)
			if err != nil {
				return completionHandlerError(err)
			}
			var handle string
			if user.Login != "" {
				handle = user.Login
			} else {
				// "author:John Doe" does not work yet, so use the first name.
				handle = strings.Split(user.Name, " ")[0]
			}
			completions[i] = handle + "\t" + user.DisplayName()
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}
