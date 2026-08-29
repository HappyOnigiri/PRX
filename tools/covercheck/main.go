package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	minimumsPath           = "tools/covercheck/minimums.txt"
	driftLimit             = 1.0
	coverageCommandTimeout = 5 * time.Minute
)

var coveragePattern = regexp.MustCompile(`coverage:[[:space:]]+([0-9]+([.][0-9]+)?)%[[:space:]]+of[[:space:]]+statements`)

type minimum struct {
	Package string
	Value   float64
}

type coverageResult struct {
	Package string
	Actual  float64
	Minimum float64
}

type coverageIssue int

const (
	coverageOK coverageIssue = iota
	coverageBelowMinimum
	coverageDrift
)

func main() {
	update := flag.Bool("update", false, "update coverage minimums to measured values")
	flag.Parse()

	if err := run(*update); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(update bool) error {
	data, err := os.ReadFile(minimumsPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", minimumsPath, err)
	}

	minimums, err := parseMinimums(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", minimumsPath, err)
	}

	results := make([]coverageResult, 0, len(minimums))
	for _, entry := range minimums {
		actual, err := measureCoverage(entry.Package)
		if err != nil {
			return err
		}
		results = append(results, coverageResult{
			Package: entry.Package,
			Actual:  actual,
			Minimum: entry.Value,
		})
	}

	if update {
		return updateMinimums(results)
	}

	failed := false
	for _, result := range results {
		fmt.Printf("%s: measured %.1f%%; minimum %.1f%%\n", result.Package, result.Actual, result.Minimum)
		switch coverageIssueFor(result.Actual, result.Minimum) {
		case coverageOK:
			// The measured coverage satisfies the configured minimum.
		case coverageBelowMinimum:
			fmt.Printf("  %s: measured %.1f%% is below minimum %.1f%%\n", result.Package, result.Actual, result.Minimum)
			failed = true
		case coverageDrift:
			difference := result.Actual - result.Minimum
			fmt.Printf("  %s: measured %.1f%% exceeds minimum %.1f%% by %.1fpt; run `go run ./tools/covercheck -update` to raise it\n", result.Package, result.Actual, result.Minimum, difference)
			failed = true
		}
	}

	if failed {
		return errors.New("go coverage check failed")
	}
	return nil
}

func parseMinimums(data []byte) ([]minimum, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	minimums := make([]minimum, 0)
	seen := make(map[string]struct{})
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("line %d must contain a package path and minimum", lineNumber)
		}
		if _, ok := seen[fields[0]]; ok {
			return nil, fmt.Errorf("line %d repeats package %q", lineNumber, fields[0])
		}

		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
			return nil, fmt.Errorf("line %d has invalid minimum %q", lineNumber, fields[1])
		}

		minimums = append(minimums, minimum{Package: fields[0], Value: value})
		seen[fields[0]] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan input: %w", err)
	}
	if len(minimums) == 0 {
		return nil, errors.New("no coverage minimums found")
	}

	return minimums, nil
}

func coverageIssueFor(actual, minimum float64) coverageIssue {
	if actual < minimum {
		return coverageBelowMinimum
	}
	if actual-minimum > driftLimit {
		return coverageDrift
	}
	return coverageOK
}

func measureCoverage(packagePath string) (float64, error) {
	profile, err := os.CreateTemp("", "prx-cover-*.out")
	if err != nil {
		return 0, fmt.Errorf("create coverage profile for %s: %w", packagePath, err)
	}
	profilePath := profile.Name()
	defer func() { _ = os.Remove(profilePath) }()
	if err := profile.Close(); err != nil {
		return 0, fmt.Errorf("close coverage profile for %s: %w", packagePath, err)
	}

	goPackage := packagePath
	if !strings.HasPrefix(goPackage, "./") {
		goPackage = "./" + goPackage
	}
	ctx, cancel := context.WithTimeout(context.Background(), coverageCommandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-count=1", "-cover", "-coverprofile="+profilePath, goPackage)
	output, err := command.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("coverage test failed for %s: %w\n%s", packagePath, err, strings.TrimSpace(string(output)))
	}

	matches := coveragePattern.FindStringSubmatch(string(output))
	if len(matches) != 3 {
		return 0, fmt.Errorf("coverage test for %s did not report statement coverage", packagePath)
	}
	actual, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse coverage for %s: %w", packagePath, err)
	}
	return actual, nil
}

func updateMinimums(results []coverageResult) error {
	var builder strings.Builder
	for _, result := range results {
		_, _ = fmt.Fprintf(&builder, "%s %.1f\n", result.Package, math.Floor(result.Actual*10)/10)
	}
	if err := os.WriteFile(minimumsPath, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", minimumsPath, err)
	}

	for _, result := range results {
		fmt.Printf("%s: minimum updated to %.1f%%\n", result.Package, math.Floor(result.Actual*10)/10)
	}
	return nil
}
