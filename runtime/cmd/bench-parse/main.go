/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"text/tabwriter"

	"github.com/onflow/cadence/runtime/parser"
)

func main() {
	if len(os.Args) <= 1 {
		data, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			panic(err)
		}
		print(bench(data).String())
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	for i := 1; i < len(os.Args); i++ {
		filename := os.Args[i]
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}

		result := bench(data)
		_, err = fmt.Fprintf(w, "%s:\t%s\n", filename, result)
		if err != nil {
			panic(err)
		}
	}

	err := w.Flush()
	if err != nil {
		panic(err)
	}
}

func bench(data []byte) testing.BenchmarkResult {
	code := string(data)

	return testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := parser.ParseProgram(code)
			if err != nil {
				panic(err)
			}
		}
	})
}
