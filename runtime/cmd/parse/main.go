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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"text/tabwriter"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
)

var benchFlag = flag.Bool("bench", false, "benchmark")

func main() {
	flag.Parse()
	args := flag.Args()
	run(args, *benchFlag)
}

func run(args []string, bench bool) {

	if len(args) == 0 {
		data, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			panic(err)
		}

		code := string(data)

		parse := func() (program *ast.Program, err error) {
			program, err = parser2.ParseProgram(code)
			return
		}

		if bench {
			result := benchParse(parse)

			_, err = fmt.Printf("%s\n", result)
			if err != nil {
				panic(err)
			}
		} else {
			result, err := parse()
			if err != nil {
				panic(err)
			}
			j, err := json.MarshalIndent(result, "", "    ")
			if err != nil {
				panic(err)
			}
			println(string(j))
		}

	} else {
		w := newTabWriter()

		for i := 0; i < len(args); i++ {
			filename := args[i]
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			code := string(data)

			parse := func() (program *ast.Program, err error) {
				program, err = parser2.ParseProgram(code)
				return
			}

			if bench {
				result := benchParse(parse)

				_, err = fmt.Fprintf(w, "%s:\t%s\n", filename, result)
				if err != nil {
					panic(err)
				}
			} else {
				result, err := parse()
				if err != nil {
					panic(err)
				}
				j, err := json.MarshalIndent(result, "", "    ")
				if err != nil {
					panic(err)
				}
				println(string(j))
			}
		}

		err := w.Flush()
		if err != nil {
			panic(err)
		}
	}

}

func benchParse(parse func() (program *ast.Program, err error)) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := parse()
			if err != nil {
				panic(err)
			}
		}
	})
}

func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
}
