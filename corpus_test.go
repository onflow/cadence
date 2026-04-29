package format_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/janezpodhostnik/cadencefmt/internal/format"
	"github.com/janezpodhostnik/cadencefmt/internal/format/verify"
)

func TestCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping corpus tests in short mode")
	}

	root := findRepoRoot(t)
	corpusDir := filepath.Join(root, "testdata", "corpus")

	if _, err := os.Stat(corpusDir); os.IsNotExist(err) {
		t.Skip("corpus not checked out; run: git submodule update --init")
	}

	var files []string
	err := filepath.WalkDir(corpusDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".cdc" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking corpus dir: %v", err)
	}

	if len(files) == 0 {
		t.Skip("no .cdc files found in corpus")
	}

	for _, path := range files {
		rel, _ := filepath.Rel(corpusDir, path)
		t.Run(rel, func(t *testing.T) {
			t.Parallel()

			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}

			// Format must succeed
			formatted, err := format.Format(src, rel, format.Default())
			if err != nil {
				t.Fatalf("format error: %v", err)
			}

			// Idempotence: format twice, compare
			second, err := format.Format(formatted, rel, format.Default())
			if err != nil {
				t.Fatalf("second format error: %v", err)
			}
			if string(formatted) != string(second) {
				t.Errorf("not idempotent.\n--- first ---\n%s\n--- second ---\n%s",
					string(formatted), string(second))
			}

			// Round-trip: AST of formatted output matches original
			if err := verify.RoundTrip(src, formatted); err != nil {
				t.Errorf("round-trip failed: %v", err)
			}

			// Comment preservation
			inputComments := commentTexts(src)
			outputComments := commentTexts(formatted)
			if len(inputComments) > 0 {
				sort.Strings(inputComments)
				sort.Strings(outputComments)
				if strings.Join(inputComments, "\n") != strings.Join(outputComments, "\n") {
					t.Errorf("comment preservation failed.\ninput comments:  %v\noutput comments: %v",
						inputComments, outputComments)
				}
			}
		})
	}
}
