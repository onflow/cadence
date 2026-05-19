/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
