package commands

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	t.Helper()

	const (
		OSArch = runtime.GOARCH + "/" + runtime.GOOS
	)

	GitCommit = "36ff9c93"
	GitLastTag = "v1.2.3"
	GitExactTag = "undefined"

	root := NewRootCommand()
	root.PersistentPreRun(root, []string{})

	testcases := []struct {
		name   string
		opts   versionOptions
		expOut string
		expErr string
	}{
		{
			name:   "default (no option)",
			opts:   versionOptions{},
			expOut: root.Version,
			expErr: rootCommandName + " version: \n",
		},
		{
			name:   "number option",
			opts:   versionOptions{number: true},
			expOut: root.Version,
			expErr: "\n",
		},
		{
			name:   "commit option",
			opts:   versionOptions{commit: true},
			expOut: GitCommit,
			expErr: "\n",
		},
		{
			name:   "all option",
			opts:   versionOptions{all: true},
			expOut: root.Version + "\n" + OSArch + "\n" + runtime.Version() + "\n",
			expErr: "git-bug version: System version: Golang version: ",
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.name, func(t *testing.T) {
			testEnv := newTestEnv(t)

			runVersion(testEnv.env, testcase.opts, root)

			require.Equal(t, testcase.expOut, testEnv.out.String())
			require.Equal(t, testcase.expErr, testEnv.err.String())
		})
	}
}
