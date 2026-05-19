package formatter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onflow/cadence/formatter"
)

// FuzzFormat feeds arbitrary bytes and asserts no panics.
// Parse errors are expected and ignored.
func FuzzFormat(f *testing.F) {
	// Seed with snapshot test inputs
	seedFromTestdata(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic on any input
		_, _ = formatter.Format(data, "fuzz.cdc", formatter.Default())
	})
}

// FuzzRoundtrip feeds bytes that parse successfully and asserts
// idempotence (format twice, compare).
func FuzzRoundtrip(f *testing.F) {
	seedFromTestdata(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		first, err := formatter.Format(data, "fuzz.cdc", formatter.Default())
		if err != nil {
			return // parse errors are fine
		}

		opts := formatter.Default()
		opts.SkipVerify = true // already verified in first pass
		second, err := formatter.Format(first, "fuzz.cdc", opts)
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
	root := findRepoRoot(f)
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
