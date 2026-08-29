// Package main fails when a target Go package contains a function that tests
// never execute.
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type coverageFunction struct {
	file string
	line int
	name string
}

type parsedCoverageFunction struct {
	coverageFunction
	percentage string
}

func (f coverageFunction) String() string {
	return fmt.Sprintf("%s:%d %s", f.file, f.line, f.name)
}

func main() {
	repositoryRoot, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to determine repository root: %v\n", err)
		os.Exit(1)
	}
	os.Exit(run(context.Background(), repositoryRoot, os.Args[1:], os.Stdout))
}

func run(ctx context.Context, repositoryRoot string, packages []string, output io.Writer) int {
	if len(packages) == 0 {
		_, _ = io.WriteString(output, "usage: checkgozerocoverage <package> [...package]\n")
		return 1
	}

	profile, err := os.CreateTemp("", "prx-go-coverage-*.out")
	if err != nil {
		_, _ = fmt.Fprintf(output, "failed to create coverage profile: %v\n", err)
		return 1
	}
	profilePath := profile.Name()
	defer func() { _ = os.Remove(profilePath) }()
	if err := profile.Close(); err != nil {
		_, _ = fmt.Fprintf(output, "failed to close coverage profile: %v\n", err)
		return 1
	}

	coverPackage := strings.Join(packages, ",")
	testArgs := []string{
		"test",
		"-count=1",
		"-coverpkg=" + coverPackage,
		"-coverprofile=" + profilePath,
	}
	testArgs = append(testArgs, packages...)
	testCommand := exec.CommandContext(ctx, "go", testArgs...)
	testCommand.Dir = repositoryRoot
	testCommand.Stdout = output
	testCommand.Stderr = output
	if err := testCommand.Run(); err != nil {
		_, _ = fmt.Fprintf(output, "go test failed: %v\n", err)
		return 1
	}

	coverOutput := bytes.NewBuffer(nil)
	coverCommand := exec.CommandContext(ctx, "go", "tool", "cover", "-func="+profilePath)
	coverCommand.Dir = repositoryRoot
	coverCommand.Stdout = coverOutput
	coverCommand.Stderr = output
	if err := coverCommand.Run(); err != nil {
		_, _ = fmt.Fprintf(output, "go tool cover failed: %v\n", err)
		return 1
	}

	zeroCoverage, err := parseCoverageOutput(coverOutput.String())
	if err != nil {
		_, _ = fmt.Fprintf(output, "failed to parse go tool cover output: %v\n", err)
		return 1
	}
	if len(zeroCoverage) == 0 {
		_, _ = io.WriteString(output, "Go zero-coverage check passed.\n")
		return 0
	}

	_, _ = io.WriteString(output, "Go functions with 0.0% coverage:\n")
	for _, function := range zeroCoverage {
		_, _ = fmt.Fprintf(output, "%s\n", function)
	}
	return 1
}

func parseCoverageOutput(value string) ([]coverageFunction, error) {
	scanner := bufio.NewScanner(strings.NewReader(value))
	scanner.Buffer(make([]byte, 1024), 10*1024*1024)
	functions := make([]parsedCoverageFunction, 0)
	totalFound := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "total:" {
			if len(fields) != 3 || fields[1] != "(statements)" || !validCoveragePercentage(fields[2]) {
				return nil, fmt.Errorf("invalid total line %q", line)
			}
			totalFound = true
			continue
		}
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid line %q", line)
		}
		file, lineNumber, err := parseCoverageLocation(fields[0])
		if err != nil {
			return nil, err
		}
		if !validCoveragePercentage(fields[2]) {
			return nil, fmt.Errorf("invalid percentage %q in line %q", fields[2], line)
		}
		functions = append(functions, parsedCoverageFunction{
			coverageFunction: coverageFunction{file: file, line: lineNumber, name: fields[1]},
			percentage:       fields[2],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read output: %w", err)
	}
	if !totalFound {
		return nil, fmt.Errorf("missing total line")
	}
	if len(functions) == 0 {
		return nil, fmt.Errorf("no function rows")
	}

	sort.Slice(functions, func(i, j int) bool {
		if functions[i].file != functions[j].file {
			return functions[i].file < functions[j].file
		}
		if functions[i].line != functions[j].line {
			return functions[i].line < functions[j].line
		}
		return functions[i].name < functions[j].name
	})
	zeroCoverage := make([]coverageFunction, 0)
	for _, function := range functions {
		if function.percentage == "0.0%" {
			zeroCoverage = append(zeroCoverage, function.coverageFunction)
		}
	}
	return zeroCoverage, nil
}

func parseCoverageLocation(value string) (string, int, error) {
	value = strings.TrimSuffix(value, ":")
	separator := strings.LastIndexByte(value, ':')
	if separator <= 0 || separator == len(value)-1 {
		return "", 0, fmt.Errorf("invalid location %q", value)
	}
	line, err := strconv.Atoi(value[separator+1:])
	if err != nil || line < 1 {
		return "", 0, fmt.Errorf("invalid location %q", value)
	}
	return value[:separator], line, nil
}

func validCoveragePercentage(value string) bool {
	if !strings.HasSuffix(value, "%") {
		return false
	}
	number := strings.TrimSuffix(value, "%")
	if number == "" || strings.Count(number, ".") != 1 {
		return false
	}
	for _, character := range number {
		if character != '.' && (character < '0' || character > '9') {
			return false
		}
	}
	_, err := strconv.ParseFloat(number, 64)
	return err == nil
}
