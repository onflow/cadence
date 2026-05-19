package formatter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onflow/cadence/formatter"
)

func loadSnapshotInputs(b *testing.B) map[string][]byte {
	b.Helper()
	root := findRepoRoot(b)
	dir := filepath.Join(root, "testdata", "format")
	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatalf("reading testdata dir: %v", err)
	}
	inputs := make(map[string][]byte, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "input.cdc"))
		if err != nil {
			b.Fatalf("reading input %s: %v", e.Name(), err)
		}
		inputs[e.Name()] = data
	}
	return inputs
}

// --- End-to-end benchmarks ---

func BenchmarkFormat_Snapshot(b *testing.B) {
	inputs := loadSnapshotInputs(b)
	opts := formatter.Default()

	var totalBytes int64
	for _, data := range inputs {
		totalBytes += int64(len(data))
	}

	b.ResetTimer()
	b.SetBytes(totalBytes)
	for b.Loop() {
		for name, data := range inputs {
			if _, err := formatter.Format(data, name+".cdc", opts); err != nil {
				b.Fatalf("format %s: %v", name, err)
			}
		}
	}
}

func BenchmarkFormat_PerCase(b *testing.B) {
	inputs := loadSnapshotInputs(b)
	opts := formatter.Default()

	for name, data := range inputs {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				if _, err := formatter.Format(data, name+".cdc", opts); err != nil {
					b.Fatalf("format: %v", err)
				}
			}
		})
	}
}
