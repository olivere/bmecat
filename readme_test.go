package bmecat_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestREADMESnippetsAreExamples fails when a Go snippet in README.md is not a
// verbatim excerpt of an example file. The compiler and `go test` check the
// examples, so a README snippet that is an excerpt of one cannot go stale.
// Indentation and blank lines are ignored, so a snippet may show the body of
// an example function at top level.
func TestREADMESnippetsAreExamples(t *testing.T) {
	examples, err := filepath.Glob("example_test.go")
	if err != nil {
		t.Fatal(err)
	}
	more, err := filepath.Glob("*/example_test.go")
	if err != nil {
		t.Fatal(err)
	}
	examples = append(examples, more...)
	sources := make(map[string][]string, len(examples))
	for _, path := range examples {
		sources[path] = normalizedLines(t, readFile(t, path))
	}

	snippets := goSnippets(t, readFile(t, "README.md"))
	if len(snippets) == 0 {
		t.Fatal("README.md has no Go snippets; is the fence detection broken?")
	}
	for _, s := range snippets {
		lines := normalizedLines(t, s.code)
		if len(lines) == 0 {
			t.Errorf("README.md:%d: Go snippet is empty", s.line)
			continue
		}
		found := false
		for _, src := range sources {
			if containsRun(src, lines) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("README.md:%d: Go snippet is not an excerpt of any of %v", s.line, examples)
		}
	}
}

// TestREADMEGoVersion fails when the Go version README.md names differs from
// the go directive in go.mod.
func TestREADMEGoVersion(t *testing.T) {
	directive := regexp.MustCompile(`(?m)^go (\S+)$`).FindStringSubmatch(readFile(t, "go.mod"))
	if directive == nil {
		t.Fatal("go.mod has no go directive")
	}
	stated := regexp.MustCompile(`(?m)^Go (\S+) or later\.$`).FindStringSubmatch(readFile(t, "README.md"))
	if stated == nil {
		t.Fatal(`README.md has no "Go <version> or later." line`)
	}
	if stated[1] != directive[1] {
		t.Errorf("README.md requires Go %s, go.mod says %s", stated[1], directive[1])
	}
}

type snippet struct {
	line int // line of the opening fence, 1-based
	code string
}

// goFence matches the opening fence of a Go code block, whatever the case of
// the language tag and whatever follows it.
var goFence = regexp.MustCompile("(?i)^(```|~~~)\\s*(go|golang)\\b")

// goSnippets returns the fenced Go blocks of a Markdown document.
func goSnippets(t *testing.T, doc string) []snippet {
	t.Helper()
	var (
		out   []snippet
		cur   *snippet
		fence string
		code  strings.Builder
	)
	sc := bufio.NewScanner(strings.NewReader(doc))
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		switch {
		case cur == nil && goFence.MatchString(line):
			cur = &snippet{line: n}
			fence = line[:3]
			code.Reset()
		case cur != nil && strings.TrimSpace(line) == fence:
			cur.code = code.String()
			out = append(out, *cur)
			cur = nil
		case cur != nil:
			code.WriteString(line)
			code.WriteByte('\n')
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if cur != nil {
		t.Fatalf("README.md:%d: unterminated Go snippet", cur.line)
	}
	return out
}

// normalizedLines returns the non-blank lines of s with surrounding
// whitespace removed.
func normalizedLines(t *testing.T, s string) []string {
	t.Helper()
	var out []string
	for line := range strings.Lines(s) {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// containsRun reports whether run appears in src as consecutive elements.
func containsRun(src, run []string) bool {
	if len(run) == 0 {
		return true
	}
	for i := range src {
		if len(src)-i >= len(run) && slices.Equal(src[i:i+len(run)], run) {
			return true
		}
	}
	return false
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
