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
	"github.com/onflow/cadence/runtime/parser2"
)

func main() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	if len(os.Args) <= 1 {
		data, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			panic(err)
		}

		oldResult, newResult := benchOldAndNew(string(data))

		_, err = fmt.Fprintf(w, "[old]\t%s\n", oldResult)
		if err != nil {
			panic(err)
		}
		_, err = fmt.Fprintf(w, "[new]\t%s\n", newResult)
		if err != nil {
			panic(err)
		}

		return
	} else {
		for i := 1; i < len(os.Args); i++ {
			filename := os.Args[i]
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}

			oldResult, newResult := benchOldAndNew(string(data))

			_, err = fmt.Fprintf(w, "%s:\t[old]\t%s\n", filename, oldResult)
			if err != nil {
				panic(err)
			}
			_, err = fmt.Fprintf(w, "%s:\t[new]\t%s\n", filename, newResult)
			if err != nil {
				panic(err)
			}
		}
	}

	err := w.Flush()
	if err != nil {
		panic(err)
	}
}

func benchOldAndNew(code string) (oldResult, newResult testing.BenchmarkResult) {
	oldResult = bench(code, func(code string) (err error) {
		_, _, err = parser.ParseProgram(code)
		return
	})

	newResult = bench(code, func(code string) (err error) {
		_, err = parser2.ParseProgram(code)
		return
	})

	return
}

func bench(code string, parse func(string) error) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := parse(code)
			if err != nil {
				panic(err)
			}
		}
	})
}
