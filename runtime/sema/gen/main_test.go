/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Go treats directories named "testdata" specially
const testDataDirectory = "testdata"

// TestFiles finds all `.cdc` files in the `testdata` directory.
// Each file turns into a test case.
// Each input file is expected to have a "golden output" file,
// with the same path, except the `.cdc` extension is replaced by `.golden.go`.
func TestFiles(t *testing.T) {

	t.Parallel()

	test := func(inputPath string) {
		// The test name is the filename without the extension.
		_, filename := filepath.Split(inputPath)
		testname := filename[:len(filename)-len(filepath.Ext(inputPath))]

		t.Run(testname, func(t *testing.T) {
			t.Parallel()

			outFile, err := os.CreateTemp(t.TempDir(), "gen.*.go")
			require.NoError(t, err)
			defer outFile.Close()

			// Do not register generated test types in base activation.
			const registerTypes = false

			gen(inputPath, outFile, registerTypes)

			goldenPath := filepath.Join(testDataDirectory, testname+".golden.go")
			want, err := os.ReadFile(goldenPath)
			require.NoError(t, err)

			_, err = outFile.Seek(0, io.SeekStart)
			require.NoError(t, err)

			got, err := io.ReadAll(outFile)
			require.NoError(t, err)

			require.Equal(t, string(want), string(got))
		})
	}

	paths, err := filepath.Glob(filepath.Join(testDataDirectory, "*.cdc"))
	require.NoError(t, err)

	for _, path := range paths {
		test(path)
	}
}
