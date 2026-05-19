package format_test

import (
	"os"
	"path/filepath"
	"testing"
)

// findRepoRoot walks up from the working directory to find the repo root
// (identified by go.mod). Works with both *testing.T and *testing.F.
func findRepoRoot(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("could not find repo root (go.mod)")
		}
		dir = parent
	}
}
