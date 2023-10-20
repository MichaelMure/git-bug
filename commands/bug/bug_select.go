package bugcmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/execenv"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entities/bug"
)

func ResolveSelected(repo *cache.RepoCache, args []string) (*cache.BugCache, []string, error) {
	return _select.Resolve[*cache.BugCache](repo, bug.Typename, bug.Namespace, repo.Bugs(), args)
}

func newBugSelectCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "select BUG_ID",
		Short: "Select a bug for implicit use in future commands",
		Example: `git bug select 2f15
git bug comment
git bug status
`,
		Long: `Select a bug for implicit use in future commands.

This command allows you to omit any bug ID argument, for example:
  git bug show
instead of
  git bug show 2f153ca

The complementary command is "git bug deselect" performing the opposite operation.
`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugSelect(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	return cmd
}

func runBugSelect(env *execenv.Env, args []string) error {
	if len(args) == 0 {
		return errors.New("a bug id must be provided")
	}

	prefix := args[0]

	b, err := env.Backend.Bugs().ResolvePrefix(prefix)
	if err != nil {
		return err
	}

	err = _select.Select(env.Backend, bug.Namespace, b.Id())
	if err != nil {
		return err
	}

	env.Out.Printf("selected bug %s: %s\n", b.Id().Human(), b.Compile().Title)

	return nil
}
