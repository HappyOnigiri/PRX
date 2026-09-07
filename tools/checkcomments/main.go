// Package main checks the size of Go comment groups.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	maxCommentLines = 3
	maxDisplayWidth = 200
	tabWidth        = 8
)

// checkedDirectories lists the trees that hold hand-written Go code. Repository-root files are
// added separately because walking "." would also descend into gen/ and web/.
var checkedDirectories = []string{"cmd", "internal", "tools"}

// fullWidthPattern keeps the same East Asian Width W/F ranges the Markdown check uses. It is split
// into several literals so that each source line stays inside the line-length limit.
const fullWidthPattern = "[ᄀ-ᅟ⌚-⌛〈-〉⏩-⏬⏰⏳◽-◾☔-☕♈-♓♿⚓⚡⚪-⚫⚽-⚾⛄-⛅⛎⛔⛪⛲-⛳⛵⛺⛽" +
	"✅✊-✋✨❌❎❓-❕❗➕-➗➰➿⬛-⬜⭐⭕⺀-⺙⺛-⻳⼀-⿕⿰-⿻　-〾ぁ-ゖ゙-ヿ" +
	"ㄅ-ㄯㄱ-ㆎ㆐-㇣ㇰ-㈞㈠-㉇㉐-㋿㌀-䶿一-꓆ꥠ-ꥼ가-힣豈-﫿" +
	"︐-︙︰-﹫！-｠￠-￦🀄🃏🆎🆑-🆚🈀-🈂🈐-🈻🉀-🉈🉐-🉑" +
	"🌀-🙏🚀-🛿🤀-🧿🩰-🫿𠀀-𿿽]"

var (
	fullWidth = regexp.MustCompile(fullWidthPattern)
	marker    = regexp.MustCompile(`^commentlint:allow-long -- (.+)$`)
	directive = regexp.MustCompile(
		`^(go:[a-zA-Z0-9_]+|[ \t]*\+build(?:\s|$)|line\s|nolint(?::[a-zA-Z0-9,]+)?(?:\s|$))`,
	)
)

type violation struct {
	path   string
	line   int
	rule   string
	actual int
}

type bodyLine struct {
	text      string
	line      int
	directive bool
}

// displayWidth counts terminal columns: tabs advance to the next tab stop and full-width runes
// take two columns.
func displayWidth(text string) int {
	width := 0
	for _, r := range text {
		switch {
		case r == '\t':
			width += tabWidth - width%tabWidth
		case fullWidth.MatchString(string(r)):
			width += 2
		default:
			width++
		}
	}
	return width
}

// commentLines strips the comment delimiters of one group and returns the remaining body lines.
func commentLines(fset *token.FileSet, group *ast.CommentGroup) []bodyLine {
	var result []bodyLine
	for _, comment := range group.List {
		block := strings.HasPrefix(comment.Text, "/*")
		raw := comment.Text[2:]
		if block {
			raw = strings.TrimSuffix(raw, "*/")
		}
		lines := strings.Split(raw, "\n")
		for i, line := range lines {
			text := strings.TrimLeft(line, " \t\r")
			if block {
				text = strings.TrimPrefix(text, "*")
				text = strings.TrimLeft(text, " \t\r")
				// A block comment that opens or closes on its own line contributes no body there.
				if strings.TrimSpace(text) == "" && (i == 0 || i == len(lines)-1) {
					continue
				}
			}
			result = append(result, bodyLine{
				text:      text,
				line:      fset.PositionFor(comment.Slash, false).Line + i,
				directive: !block && directive.MatchString(raw),
			})
		}
	}
	return result
}

// checkGroup reports the violations of one comment group. Directive lines count towards neither
// limit, and a valid marker waives the line limit only.
func checkGroup(path string, lines []bodyLine) []violation {
	var issues []violation
	var body []bodyLine
	markers := 0
	allowed := false
	for _, line := range lines {
		switch {
		case line.directive:
			continue
		case strings.Contains(line.text, "commentlint:"):
			markers++
			match := marker.FindStringSubmatch(line.text)
			valid := len(match) == 2 && strings.TrimSpace(match[1]) != ""
			if !valid {
				issues = append(issues, violation{path, line.line, "marker-format", markers})
			}
			if markers > 1 {
				issues = append(issues, violation{path, line.line, "marker-duplicate", markers})
			}
			allowed = allowed || valid
		default:
			body = append(body, line)
		}
		if width := displayWidth(line.text); width > maxDisplayWidth {
			issues = append(issues, violation{path, line.line, "width", width})
		}
	}
	for len(body) > 0 && strings.TrimSpace(body[0].text) == "" {
		body = body[1:]
	}
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1].text) == "" {
		body = body[:len(body)-1]
	}
	if markers > 0 && len(body) == 0 {
		issues = append(issues, violation{path, lines[0].line, "marker-empty", 0})
	}
	lineCount := 0
	previousLine := 0
	for _, line := range body {
		if line.line != previousLine {
			lineCount++
			previousLine = line.line
		}
	}
	if !allowed && lineCount > maxCommentLines {
		issues = append(issues, violation{path, body[0].line, "lines", lineCount})
	}
	return issues
}

func checkFile(path string) ([]violation, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}
	if ast.IsGenerated(file) {
		return nil, nil
	}
	var issues []violation
	for _, group := range file.Comments {
		issues = append(issues, checkGroup(path, commentLines(fset, group))...)
	}
	return issues, nil
}

func collectFile(path string, issues *[]violation, failures *[]string) {
	found, err := checkFile(path)
	if err != nil {
		*failures = append(*failures, fmt.Sprintf("%s: parse/read: %v", path, err))
	}
	*issues = append(*issues, found...)
}

func run(root string, out io.Writer) int {
	var issues []violation
	var failures []string
	for _, directory := range checkedDirectories {
		err := filepath.WalkDir(
			filepath.Join(root, directory),
			func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || filepath.Ext(path) != ".go" {
					return nil
				}
				collectFile(path, &issues, &failures)
				return nil
			},
		)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}
	rootFiles, err := filepath.Glob(filepath.Join(root, "*.go"))
	if err != nil {
		failures = append(failures, err.Error())
	}
	for _, path := range rootFiles {
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		collectFile(path, &issues, &failures)
	}
	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.path != b.path {
			return a.path < b.path
		}
		if a.line != b.line {
			return a.line < b.line
		}
		return a.rule < b.rule
	})
	for _, issue := range issues {
		_, _ = fmt.Fprintf(out, "%s:%d: %s: actual=%d\n", issue.path, issue.line, issue.rule, issue.actual)
	}
	sort.Strings(failures)
	for _, failure := range failures {
		_, _ = fmt.Fprintln(out, failure)
	}
	if len(issues)+len(failures) > 0 {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(".", os.Stderr))
}
