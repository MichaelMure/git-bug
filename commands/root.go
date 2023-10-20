// Package commands contains the CLI commands
package commands

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	boardcmd "github.com/MichaelMure/git-bug/commands/board"
	bridgecmd "github.com/MichaelMure/git-bug/commands/bridge"
	bugcmd "github.com/MichaelMure/git-bug/commands/bug"
	"github.com/MichaelMure/git-bug/commands/execenv"
	usercmd "github.com/MichaelMure/git-bug/commands/user"
)

// These variables are initialized externally during the build. See the Makefile.
var GitCommit string
var GitLastTag string
var GitExactTag string

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   execenv.RootCommandName,
		Short: "A bug tracker embedded in Git",
		Long: `git-bug is a bug tracker embedded in git.

git-bug use git objects to store the bug tracking separated from the files
history. As bugs are regular git objects, they can be pushed and pulled from/to
the same git remote you are already using to collaborate with other people.

`,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			root := cmd.Root()
			root.Version = getVersion()
		},

		// For the root command, force the execution of the PreRun
		// even if we just display the help. This is to make sure that we check
		// the repository and give the user early feedback.
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				os.Exit(1)
			}
		},

		SilenceUsage:      true,
		DisableAutoGenTag: true,
	}

	const entityGroup = "entity"
	const uiGroup = "ui"
	const remoteGroup = "remote"

	cmd.AddGroup(&cobra.Group{ID: entityGroup, Title: "Entities"})
	cmd.AddGroup(&cobra.Group{ID: uiGroup, Title: "Interactive interfaces"})
	cmd.AddGroup(&cobra.Group{ID: remoteGroup, Title: "Interaction with the outside world"})

	addCmdWithGroup := func(child *cobra.Command, groupID string) {
		cmd.AddCommand(child)
		child.GroupID = groupID
	}

	env := execenv.NewEnv()

	addCmdWithGroup(boardcmd.NewBoardCommand(), entityGroup)
	addCmdWithGroup(bugcmd.NewBugCommand(env), entityGroup)
	addCmdWithGroup(usercmd.NewUserCommand(env), entityGroup)
	addCmdWithGroup(newLabelCommand(env), entityGroup)

	addCmdWithGroup(newTermUICommand(env), uiGroup)
	addCmdWithGroup(newWebUICommand(env), uiGroup)

	addCmdWithGroup(newPullCommand(env), remoteGroup)
	addCmdWithGroup(newPushCommand(env), remoteGroup)
	addCmdWithGroup(bridgecmd.NewBridgeCommand(env), remoteGroup)

	cmd.AddCommand(newCommandsCommand(env))
	cmd.AddCommand(newVersionCommand(env))
	cmd.AddCommand(newWipeCommand(env))

	return cmd
}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func getVersion() string {
	if GitExactTag == "undefined" {
		GitExactTag = ""
	}

	if GitExactTag != "" {
		// we are exactly on a tag --> release version
		return GitLastTag
	}

	if GitLastTag != "" {
		// not exactly on a tag --> dev version
		return fmt.Sprintf("%s-dev-%.10s", GitLastTag, GitCommit)
	}

	// we don't have commit information, try golang build info
	if commit, dirty, err := getCommitAndDirty(); err == nil {
		if dirty {
			return fmt.Sprintf("dev-%.10s-dirty", commit)
		}
		return fmt.Sprintf("dev-%.10s", commit)
	}

	return "dev-unknown"
}

func getCommitAndDirty() (commit string, dirty bool, err error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false, fmt.Errorf("unable to read build info")
	}

	var commitFound bool

	// get the commit and modified status
	// (that is the flag for repository dirty or not)
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			commit = kv.Value
			commitFound = true
		case "vcs.modified":
			if kv.Value == "true" {
				dirty = true
			}
		}
	}

	if !commitFound {
		return "", false, fmt.Errorf("no commit found")
	}

	return commit, dirty, nil
}
