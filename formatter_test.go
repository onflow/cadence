package format_test

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/janezpodhostnik/cadencefmt/internal/format"
	"github.com/janezpodhostnik/cadencefmt/internal/format/trivia"
	"github.com/janezpodhostnik/cadencefmt/internal/format/verify"
)

var update = flag.Bool("update", false, "update golden files")

func TestSnapshot(t *testing.T) {
	t.Parallel()
	testdataDir := filepath.Join(findRepoRoot(t), "testdata", "format")

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(testdataDir, name)
			inputPath := filepath.Join(dir, "input.cdc")
			goldenPath := filepath.Join(dir, "golden.cdc")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("reading input: %v", err)
			}

			got, err := format.Format(input, inputPath, format.Default())
			if err != nil {
				t.Fatalf("format error: %v", err)
			}

			if *update {
				if err := os.WriteFile(goldenPath, got, 0644); err != nil {
					t.Fatalf("writing golden: %v", err)
				}
				return
			}

			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden (run with -update to create): %v", err)
			}

			if string(got) != string(golden) {
				t.Errorf("output does not match golden.\n--- got ---\n%s\n--- golden ---\n%s",
					string(got), string(golden))
			}
		})
	}
}

func TestIdempotence(t *testing.T) {
	t.Parallel()
	testdataDir := filepath.Join(findRepoRoot(t), "testdata", "format")

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(testdataDir, name)
			inputPath := filepath.Join(dir, "input.cdc")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("reading input: %v", err)
			}

			first, err := format.Format(input, inputPath, format.Default())
			if err != nil {
				t.Fatalf("first format: %v", err)
			}

			second, err := format.Format(first, inputPath, format.Default())
			if err != nil {
				t.Fatalf("second format: %v", err)
			}

			if string(first) != string(second) {
				t.Errorf("not idempotent.\n--- first ---\n%s\n--- second ---\n%s",
					string(first), string(second))
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	testdataDir := filepath.Join(findRepoRoot(t), "testdata", "format")

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(testdataDir, name)
			inputPath := filepath.Join(dir, "input.cdc")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("reading input: %v", err)
			}

			output, err := format.Format(input, inputPath, format.Default())
			if err != nil {
				t.Fatalf("format error: %v", err)
			}

			if err := verify.RoundTrip(input, output); err != nil {
				t.Errorf("round-trip failed: %v", err)
			}
		})
	}
}

func TestCommentPreservation(t *testing.T) {
	t.Parallel()
	testdataDir := filepath.Join(findRepoRoot(t), "testdata", "format")

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(testdataDir, name)
			inputPath := filepath.Join(dir, "input.cdc")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("reading input: %v", err)
			}

			output, err := format.Format(input, inputPath, format.Default())
			if err != nil {
				t.Fatalf("format error: %v", err)
			}

			// Extract comment texts from input and output
			inputComments := commentTexts(input)
			outputComments := commentTexts(output)

			if len(inputComments) == 0 {
				return // no comments to check
			}

			// Compare as sorted multisets
			sort.Strings(inputComments)
			sort.Strings(outputComments)

			if strings.Join(inputComments, "\n") != strings.Join(outputComments, "\n") {
				t.Errorf("comment preservation failed.\ninput comments:  %v\noutput comments: %v",
					inputComments, outputComments)
			}
		})
	}
}

func TestKeepBlankLines_Zero(t *testing.T) {
	t.Parallel()
	src := []byte("access(all) fun a() {}\n\n\naccess(all) fun b() {}\n")
	opts := format.Default()
	opts.KeepBlankLines = 0
	got, err := format.Format(src, "test.cdc", opts)
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if strings.Contains(string(got), "\n\n") {
		t.Errorf("expected no blank lines with KeepBlankLines=0, got:\n%s", got)
	}
}

func TestKeepBlankLines_Two(t *testing.T) {
	t.Parallel()
	src := []byte("access(all) fun a() {}\n\n\n\n\naccess(all) fun b() {}\n")
	opts := format.Default()
	opts.KeepBlankLines = 2
	got, err := format.Format(src, "test.cdc", opts)
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if strings.Contains(string(got), "\n\n\n\n") {
		t.Errorf("expected at most 2 blank lines with KeepBlankLines=2, got:\n%s", got)
	}
}

func TestKeepBlankLines_Default(t *testing.T) {
	t.Parallel()
	src := []byte("access(all) fun a() {}\n\n\n\n\naccess(all) fun b() {}\n")
	got, err := format.Format(src, "test.cdc", format.Default())
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if strings.Contains(string(got), "\n\n\n") {
		t.Errorf("expected at most 1 blank line with default options, got:\n%s", got)
	}
}

func TestStripSemicolons_Default(t *testing.T) {
	t.Parallel()
	src := []byte("access(all) let x: Int = 1;\n")
	got, err := format.Format(src, "test.cdc", format.Default())
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if strings.Contains(string(got), ";") {
		t.Errorf("expected semicolons stripped by default, got:\n%s", got)
	}
}

func TestStripSemicolons_False(t *testing.T) {
	t.Parallel()
	src := []byte("access(all) let x: Int = 1;\n")
	opts := format.Default()
	opts.StripSemicolons = false
	got, err := format.Format(src, "test.cdc", opts)
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if !strings.Contains(string(got), ";") {
		t.Errorf("expected semicolons preserved with StripSemicolons=false, got:\n%s", got)
	}
}

func TestStripSemicolons_Idempotent(t *testing.T) {
	t.Parallel()
	src := []byte("access(all) let x: Int = 1;\n")
	opts := format.Default()
	opts.StripSemicolons = false
	first, err := format.Format(src, "test.cdc", opts)
	if err != nil {
		t.Fatalf("first format error: %v", err)
	}
	second, err := format.Format(first, "test.cdc", opts)
	if err != nil {
		t.Fatalf("second format error: %v", err)
	}
	if string(first) != string(second) {
		t.Errorf("not idempotent with StripSemicolons=false.\n--- first ---\n%s\n--- second ---\n%s",
			first, second)
	}
}

func TestFormatVersion_Unsupported(t *testing.T) {
	t.Parallel()
	opts := format.Default()
	opts.FormatVersion = "99"
	_, err := format.Format([]byte("access(all) fun main() {}"), "test.cdc", opts)
	if err == nil {
		t.Fatal("expected error for unsupported format version")
	}
	if !strings.Contains(err.Error(), "unsupported format version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatVersion_Current(t *testing.T) {
	t.Parallel()
	_, err := format.Format([]byte("access(all) fun main() {}"), "test.cdc", format.Default())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func commentTexts(src []byte) []string {
	comments := trivia.Scan(src)
	texts := make([]string, len(comments))
	for i, c := range comments {
		// Normalize: strip trailing whitespace from each line within the
		// comment, so blank lines inside block comments compare equal
		// regardless of indentation whitespace.
		lines := strings.Split(c.Text, "\n")
		for j, line := range lines {
			lines[j] = strings.TrimRight(line, " \t")
		}
		texts[i] = strings.Join(lines, "\n")
	}
	return texts
}

func TestNoTrailingWhitespace(t *testing.T) {
	t.Parallel()
	testdataDir := filepath.Join(findRepoRoot(t), "testdata", "format")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input, err := os.ReadFile(filepath.Join(testdataDir, name, "input.cdc"))
			if err != nil {
				t.Fatalf("reading input: %v", err)
			}
			got, err := format.Format(input, "test.cdc", format.Default())
			if err != nil {
				t.Fatalf("format error: %v", err)
			}
			for i, line := range strings.Split(string(got), "\n") {
				trimmed := strings.TrimRight(line, " \t")
				if trimmed != line {
					t.Errorf("line %d has trailing whitespace: %q", i+1, line)
				}
			}
		})
	}
}

