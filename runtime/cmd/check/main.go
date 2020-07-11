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
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/onflow/cadence/runtime/cmd"
)

var benchFlag = flag.Bool("bench", false, "benchmark the checker")
var jsonFlag = flag.Bool("json", false, "print the result formatted as JSON")

func main() {
	flag.Parse()
	args := flag.Args()
	run(args, *benchFlag, *jsonFlag)
}

type result struct {
	Path       string                  `json:",omitempty"`
	Bench      testing.BenchmarkResult `json:",omitempty"`
	CheckError string                  `json:",omitempty"`
}

type output interface {
	Append(result)
	End()
}

type jsonOutput struct {
	results []result
}

func newJSONOutput(count int) *jsonOutput {
	return &jsonOutput{
		results: make([]result, 0, count),
	}
}

func (j *jsonOutput) Append(r result) {
	j.results = append(j.results, r)
}

func (j *jsonOutput) End() {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(j.results)
	if err != nil {
		panic(err)
	}
}

type stdoutOutput struct {
	writer *tabwriter.Writer
}

func (s stdoutOutput) Append(r result) {
	var err error

	if len(r.Path) > 0 {
		_, err = fmt.Fprintf(s.writer, "%s\n", r.Path)
		if err != nil {
			panic(err)
		}
	}

	if r.Bench.N > 0 {
		_, err = fmt.Fprintf(s.writer, "bench:\t%s\n", r.Bench)
		if err != nil {
			panic(err)
		}
	}

	if r.CheckError != "" {
		_, err = fmt.Fprintf(s.writer, "error:\t%s\n", r.CheckError)
		if err != nil {
			panic(err)
		}
	}

	err = s.writer.Flush()
	if err != nil {
		panic(err)
	}
}

func (s stdoutOutput) End() {
	// no-op
}

func newStdoutOutput() stdoutOutput {
	return stdoutOutput{
		writer: tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0),
	}
}

func run(paths []string, bench bool, json bool) {
	if len(paths) == 0 {
		paths = []string{""}
	}

	var out output
	if json {
		out = newJSONOutput(len(paths))
	} else {
		out = newStdoutOutput()
	}

	for _, path := range paths {
		res := result{
			Path: path,
		}

		code := read(path)

		program, codes, must := cmd.PrepareProgram(code, path)

		checker, _ := cmd.PrepareChecker(program, path, must)

		err := checker.Check()
		if err != nil {
			var builder strings.Builder
			cmd.PrettyPrintError(&builder, err, path, codes)
			res.CheckError = builder.String()
		}

		if bench && err == nil {
			res.Bench = testing.Benchmark(func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					checker, must = cmd.PrepareChecker(program, path, must)
					must(checker.Check())
					if err != nil {
						panic(err)
					}
				}
			})
		}

		out.Append(res)
	}

	out.End()
}

func read(path string) string {
	var data []byte
	var err error
	if len(path) == 0 {
		data, err = ioutil.ReadAll(bufio.NewReader(os.Stdin))
	} else {
		data, err = ioutil.ReadFile(path)
	}
	if err != nil {
		panic(err)
	}
	return string(data)
}
