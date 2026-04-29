package format_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janezpodhostnik/cadencefmt/internal/format"
)

// FuzzFormat feeds arbitrary bytes and asserts no panics.
// Parse errors are expected and ignored.
func FuzzFormat(f *testing.F) {
	// Seed with snapshot test inputs
	seedFromTestdata(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic on any input
		_, _ = format.Format(data, "fuzz.cdc", format.Default())
	})
}

// FuzzRoundtrip feeds bytes that parse successfully and asserts
// idempotence (format twice, compare).
func FuzzRoundtrip(f *testing.F) {
	seedFromTestdata(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		first, err := format.Format(data, "fuzz.cdc", format.Default())
		if err != nil {
			return // parse errors are fine
		}

		opts := format.Default()
		opts.SkipVerify = true // already verified in first pass
		second, err := format.Format(first, "fuzz.cdc", opts)
		if err != nil {
			t.Fatalf("second format failed: %v", err)
		}

		if string(first) != string(second) {
			t.Errorf("not idempotent.\n--- first (%d bytes) ---\n%s\n--- second (%d bytes) ---\n%s",
				len(first), first, len(second), second)
		}
	})
}

func seedFromTestdata(f *testing.F) {
	f.Helper()
	root := findFuzzRepoRoot(f)
	testdataDir := filepath.Join(root, "testdata", "format")

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		inputPath := filepath.Join(testdataDir, entry.Name(), "input.cdc")
		data, err := os.ReadFile(inputPath)
		if err != nil {
			continue
		}
		f.Add(data)
	}
}

func findFuzzRepoRoot(f *testing.F) string {
	f.Helper()
	dir, err := os.Getwd()
	if err != nil {
		f.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			f.Fatal("could not find repo root")
		}
		dir = parent
	}
}
