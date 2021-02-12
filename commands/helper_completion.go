package commands

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	_select "github.com/MichaelMure/git-bug/commands/select"
)

type validArgsFunction func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective)

func completionHandlerError(err error) (completions []string, directives cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveError
}

func completeBridge(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
		}()

		bridges, err := bridge.ConfiguredBridges(env.backend)
		if err != nil {
			return completionHandlerError(err)
		}

		completions = make([]string, len(bridges))
		for i, bridge := range bridges {
			completions[i] = bridge + "\t" + "Bridge"
		}

		return completions, cobra.ShellCompDirectiveDefault
	}
}

func completeBridgeAuth(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
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

		return completions, cobra.ShellCompDirectiveDefault
	}
}

func completeBug(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
		}()

		allIds := env.backend.AllBugsIds()
		bugExcerpt := make([]*cache.BugExcerpt, len(allIds))
		for i, id := range allIds {
			var err error
			bugExcerpt[i], err = env.backend.ResolveBugExcerpt(id)
			if err != nil {
				return completionHandlerError(err)
			}
		}

		completions = make([]string, len(allIds))
		for i, id := range allIds {
			completions[i] = id.Human() + "\t" + bugExcerpt[i].Title
		}
		return completions, cobra.ShellCompDirectiveDefault
	}
}

func completeBugAndLabels(env *Env, addOrRemove bool) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeBug(env)(cmd, args, toComplete)
		}

		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
		}()

		b, args, err := _select.ResolveBug(env.backend, args)
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

		return completions, cobra.ShellCompDirectiveDefault
	}
}

func completeFrom(choices []string) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return choices, cobra.ShellCompDirectiveDefault
	}
}

func completeGitRemote(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
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
		return completions, cobra.ShellCompDirectiveDefault
	}
}

func completeLabel(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
		}()

		labels := env.backend.ValidLabels()
		completions = make([]string, len(labels))
		for i, label := range labels {
			completions[i] = string(label) + "\t" + "Label"
		}
		return completions, cobra.ShellCompDirectiveDefault
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
				_ = closeBackend(env)(cmd, args)
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
			return completions, cobra.ShellCompDirectiveDefault
		}

		for _, key := range byLabel {
			if !strings.HasPrefix(toComplete, key) {
				continue
			}
			labels := env.backend.ValidLabels()
			completions = make([]string, len(labels))
			for i, label := range labels {
				completions[i] = key + string(label)
			}
			return completions, cobra.ShellCompDirectiveDefault
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
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
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
		return completions, cobra.ShellCompDirectiveDefault
	}
}

func completeUserForQuery(env *Env) validArgsFunction {
	return func(cmd *cobra.Command, args []string, toComplete string) (completions []string, directives cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		if err := loadBackend(env)(cmd, args); err != nil {
			return completionHandlerError(err)
		}
		defer func() {
			_ = closeBackend(env)(cmd, args)
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
		return completions, cobra.ShellCompDirectiveDefault
	}
}
