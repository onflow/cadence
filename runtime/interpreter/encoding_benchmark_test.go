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

package interpreter_test

// TODO:
//
//
//import (
//	"bytes"
//	"compress/gzip"
//	"fmt"
//	"io"
//	"os"
//	"path/filepath"
//	"regexp"
//	"strconv"
//	"strings"
//	"testing"
//
//	"github.com/onflow/cadence/runtime/common"
//	"github.com/stretchr/testify/require"
//)
//
//type encodingBenchmark struct {
//	name  string
//	value Value
//}
//
//func BenchmarkEncodeCBOR(b *testing.B) {
//	benchmarks := prepareCBORTestData()
//
//	for _, bm := range benchmarks {
//		b.Run(bm.name, func(b *testing.B) {
//
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				_, _, err := EncodeValue(bm.value, nil, true, nil)
//				require.NoError(b, err)
//			}
//		})
//	}
//}
//
//func BenchmarkDecodeCBOR(b *testing.B) {
//	benchmarks := prepareCBORTestData()
//
//	for _, bm := range benchmarks {
//		b.Run(bm.name, func(b *testing.B) {
//
//			encoded, _, err := EncodeValue(bm.value, nil, true, nil)
//			require.NoError(b, err)
//
//			b.ReportAllocs()
//			b.ResetTimer()
//
//			for i := 0; i < b.N; i++ {
//				_, err = DecodeValue(encoded, nil, nil, CurrentEncodingVersion, nil)
//				require.NoError(b, err)
//			}
//		})
//	}
//}
//
//// prepareCBORTestData processes testdata/*.cbor.gz files and
//// returns a []encodingBenchmark including each encodingBenchmark name and Value.
//// For safety, the number of *.cbor.gz files processed is limited and
//// io.LimitReader is used to limit uncompressed CBOR data size.
//func prepareCBORTestData() []encodingBenchmark {
//	// maxNumFiles limits the number of files processed in testdata folder.
//	const maxNumFiles = 100
//
//	// maxCBORSize limits max size of each uncompressed CBOR data to 16 MB.
//	const maxCBORSize = 16 * 1024 * 1024
//
//	// Compile regular expression to match version number in file name
//	versionRegExp := regexp.MustCompile(`_v([\d]+)_`)
//
//	fileNames, err := filepath.Glob("testdata/*.cbor.gz")
//	if err != nil {
//		panic(err)
//	}
//
//	numFiles := len(fileNames)
//	if numFiles > maxNumFiles {
//		numFiles = maxNumFiles
//
//		fmt.Printf("Found %d *.cbor.gz files in testdata.  Processing only the first %d files.\n",
//			len(fileNames),
//			maxNumFiles)
//	}
//
//	var benchmarks []encodingBenchmark
//	for _, fileName := range fileNames[:numFiles] {
//
//		// Read cbor.gz file
//		f, err := os.Open(fileName)
//		if err != nil {
//			panic(err)
//		}
//		defer f.Close()
//
//		zr, err := gzip.NewReader(f)
//		if err != nil {
//			panic(err)
//		}
//		defer zr.Close()
//
//		lzr := io.LimitReader(zr, maxCBORSize)
//
//		var buf bytes.Buffer
//		fileSize, err := io.Copy(&buf, lzr)
//		if err != nil {
//			panic(err)
//		}
//
//		// File name without path nor extension
//		name := strings.TrimSuffix(filepath.Base(fileName), ".cbor.gz")
//
//		// Get version number from file name
//		// If file name doesn't match "_v([\d+])_", version number is default to current version.
//		version := CurrentEncodingVersion
//		m := versionRegExp.FindStringSubmatch(name)
//		if len(m) > 1 {
//			v, err := strconv.ParseUint(m[1], 10, 16)
//			if err == nil {
//				version = uint16(v)
//			}
//		}
//
//		// Construct encodingBenchmark name
//		name += fmt.Sprintf("_%dbytes", fileSize)
//
//		// Decode test data to value
//		owner := common.MustBytesToAddress([]byte{})
//
//		value, err := DecodeValue(buf.Bytes(), &owner, nil, version, nil)
//		if err != nil {
//			panic(fmt.Sprintf("failed to decode value in file %s: %s\n", fileName, err))
//		}
//
//		// Add to benchmarks
//		benchmarks = append(benchmarks, encodingBenchmark{name: name, value: value})
//	}
//
//	return benchmarks
//}
