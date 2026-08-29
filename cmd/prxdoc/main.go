package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra/doc"

	"github.com/HappyOnigiri/PRX/internal/cli"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: prxdoc OUTPUT_DIR")
		os.Exit(2)
	}
	if err := generate(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "generate CLI documentation:", err)
		os.Exit(1)
	}
}

func generate(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read output directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("remove %s: %w", entry.Name(), err)
		}
	}

	root := cli.NewRoot(io.Discard, io.Discard, nil)
	root.DisableAutoGenTag = true
	root.CompletionOptions.HiddenDefaultCmd = true
	return doc.GenMarkdownTree(root, dir)
}
