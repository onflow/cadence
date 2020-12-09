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
	"runtime/debug"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
)

var benchFlag = flag.Bool("bench", false, "benchmark the checker")
var jsonFlag = flag.Bool("json", false, "print the result formatted as JSON")

func main() {
	flag.Parse()
	args := flag.Args()
	run(args, *benchFlag, *jsonFlag)
}

type benchResult struct {
	// N is the the number of iterations
	Iterations int `json:"iterations"`
	// T is the total time taken
	Time time.Duration `json:"time"`
}

type result struct {
	Path     string       `json:"path"`
	Bench    *benchResult `json:"bench,omitempty"`
	BenchStr string       `json:"-"`
	Error    string       `json:"error,omitempty"`
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

	if len(r.BenchStr) > 0 {
		_, err = fmt.Fprintf(s.writer, "bench:\t%s\n", r.BenchStr)
		if err != nil {
			panic(err)
		}
	}

	if len(r.Error) > 0 {
		_, err = fmt.Fprintf(s.writer, "error:\t%s\n", r.Error)
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

	allSucceeded := true

	var out output
	if json {
		out = newJSONOutput(len(paths))
	} else {
		out = newStdoutOutput()
	}

	useColor := !json

	for _, path := range paths {
		res, runSucceeded := runPath(path, bench, useColor)
		if !runSucceeded {
			allSucceeded = false
		}

		out.Append(res)
	}

	out.End()

	if !allSucceeded {
		os.Exit(1)
	}
}

func runPath(path string, bench bool, useColor bool) (res result, succeeded bool) {
	res = result{
		Path: path,
	}
	succeeded = true

	code := read(path)

	var err error
	var checker *sema.Checker
	var program *ast.Program
	var must func(error)

	codes := map[string]string{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%s", debug.Stack())
				res.Error = err.Error()
			}
		}()

		program, must = cmd.PrepareProgram(code, path, codes)

		checker, _ = cmd.PrepareChecker(program, path, codes, must)

		err = checker.Check()
		if err != nil {
			var builder strings.Builder
			printErr := pretty.NewErrorPrettyPrinter(&builder, useColor).
				PrettyPrintError(err, path, codes)
			if printErr != nil {
				panic(printErr)
			}
			res.Error = builder.String()
		}
	}()

	if err != nil {
		succeeded = false
	}

	if bench && err == nil {
		benchRes := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				checker, must = cmd.PrepareChecker(program, path, codes, must)
				must(checker.Check())
				if err != nil {
					panic(err)
				}
			}
		})
		res.Bench = &benchResult{
			Iterations: benchRes.N,
			Time:       benchRes.T,
		}
		res.BenchStr = benchRes.String()
	}

	return res, succeeded
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
