package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands"
)

func main() {
	fmt.Println("Generating completion files ...")

	tasks := map[string]func(*cobra.Command) error{
		"Bash":       genBash,
		"Fish":       genFish,
		"PowerShell": genPowerShell,
		"ZSH":        genZsh,
	}

	for name, f := range tasks {
		root := commands.NewRootCommand()
		err := f(root)
		if err != nil {
			fmt.Printf("  - %s: %v\n", name, err)
			continue
		}
		fmt.Printf("  - %s: ok\n", name)
	}
}

func genBash(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "misc", "bash_completion", "git-bug")
	return root.GenBashCompletionFile(dir)
}

func genFish(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "misc", "fish_completion", "git-bug")
	return root.GenFishCompletionFile(dir, true)
}

func genPowerShell(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, "misc", "powershell_completion", "git-bug")
	return root.GenPowerShellCompletionFile(path)
}

func genZsh(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, "misc", "zsh_completion", "git-bug")
	return root.GenZshCompletionFile(path)
}
