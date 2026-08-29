// Package main checks the size of Web TypeScript files.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	warningLineLimit = 600
	hardLineLimit    = 1000
	excludedRoot     = "web/src/gen"
)

func main() {
	repositoryRoot, err := os.Getwd()
	if err != nil {
		if _, writeErr := fmt.Fprintf(os.Stderr, "failed to determine repository root: %v\n", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
	os.Exit(run(repositoryRoot, os.Stdout))
}

func run(repositoryRoot string, output io.Writer) int {
	paths, err := findTypeScriptFiles(repositoryRoot)
	if err != nil {
		if writeErr := writeOutput(
			output,
			[]string{fmt.Sprintf("failed to list Web TypeScript files: %v", err)},
		); writeErr != nil {
			return 1
		}
		return 1
	}

	warnings := make([]string, 0)
	fileErrors := make([]string, 0)
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(path)))
		if err != nil {
			fileErrors = append(fileErrors, fmt.Sprintf("%s: failed to read file: %v", path, err))
			continue
		}

		lineCount := countLines(content)
		switch {
		case lineCount > hardLineLimit:
			fileErrors = append(fileErrors, fmt.Sprintf("%s: %d lines (maximum is %d)", path, lineCount, hardLineLimit))
		case lineCount > warningLineLimit:
			warnings = append(
				warnings,
				fmt.Sprintf("%s: %d lines (warning above %d)", path, lineCount, warningLineLimit),
			)
		}
	}

	outputLines := make([]string, 0, len(warnings)+len(fileErrors)+2)
	if len(warnings) > 0 {
		outputLines = append(outputLines, "Web TypeScript file-size warnings:")
		for _, warning := range warnings {
			outputLines = append(outputLines, "- "+warning)
		}
	}
	if len(fileErrors) > 0 {
		outputLines = append(outputLines, "Web TypeScript file-size errors:")
		for _, fileError := range fileErrors {
			outputLines = append(outputLines, "- "+fileError)
		}
		if writeErr := writeOutput(output, outputLines); writeErr != nil {
			return 1
		}
		return 1
	}

	outputLines = append(outputLines, fmt.Sprintf("No Web TypeScript files exceed %d lines.", hardLineLimit))
	if writeErr := writeOutput(output, outputLines); writeErr != nil {
		return 1
	}
	return 0
}

func writeOutput(output io.Writer, lines []string) error {
	_, err := io.WriteString(output, strings.Join(lines, "\n")+"\n")
	return err
}

func findTypeScriptFiles(repositoryRoot string) ([]string, error) {
	command := exec.CommandContext(
		context.Background(),
		"git",
		"ls-files",
		"--cached",
		"--others",
		"--exclude-standard",
		"-z",
		"--",
		"web/src",
		"web/tests",
	)
	command.Dir = repositoryRoot
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	paths := make([]string, 0)
	for _, path := range strings.Split(string(output), "\x00") {
		if path == "" || !isTypeScriptFile(path) || isExcludedPath(path) {
			continue
		}
		fileInfo, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(path)))
		if err != nil || !fileInfo.Mode().IsRegular() {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func isTypeScriptFile(path string) bool {
	extension := filepath.Ext(path)
	return extension == ".ts" || extension == ".tsx"
}

func isExcludedPath(path string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	return cleanPath == excludedRoot || strings.HasPrefix(cleanPath, excludedRoot+"/")
}

func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	lineCount := strings.Count(string(content), "\n")
	if content[len(content)-1] != '\n' {
		lineCount++
	}
	return lineCount
}
