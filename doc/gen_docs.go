package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/MichaelMure/git-bug/commands"
)

func main() {
	fmt.Println("Generating documentation ...")

	tasks := map[string]func(*cobra.Command) error{
		"ManPage":  genManPage,
		"Markdown": genMarkdown,
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

func genManPage(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "doc", "man")

	// fixed date to avoid having to commit each month
	date := time.Date(2019, 4, 1, 12, 0, 0, 0, time.UTC)

	header := &doc.GenManHeader{
		Title:   "GIT-BUG",
		Section: "1",
		Date:    &date,
		Source:  "Generated from git-bug's source code",
	}

	files, err := filepath.Glob(dir + "/*.1")
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	return doc.GenManTree(root, header, dir)
}

func genMarkdown(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "doc", "md")

	files, err := filepath.Glob(dir + "/*.md")
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	return doc.GenMarkdownTree(root, dir)
}
