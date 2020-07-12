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

	"github.com/go-test/deep"

	"github.com/onflow/cadence/runtime/ast"
	parser1 "github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/parser2"
)

var benchFlag = flag.Bool("bench", false, "benchmark the new parser")
var oldFlag = flag.Bool("old", false, "also run old parser")
var compareFlag = flag.Bool("compare", false, "compare the results of the new and old parser")
var jsonFlag = flag.Bool("json", false, "print the result formatted as JSON")

func main() {
	flag.Parse()
	args := flag.Args()
	run(args, *benchFlag, *oldFlag, *compareFlag, *jsonFlag)
}

type benchResult struct {
	// N is the the number of iterations
	Iterations int `json:"iterations"`
	// T is the total time taken
	Time time.Duration `json:"time"`
}

type result struct {
	Path        string       `json:"path,omitempty"`
	BenchOld    *benchResult `json:"bench_old,omitempty"`
	BenchOldStr string       `json:"-"`
	Bench       *benchResult `json:"bench,omitempty"`
	BenchStr    string       `json:"-"`
	Diff        string       `json:"diff,omitempty"`
	Error       error        `json:"error,omitempty"`
	ErrorOld    error        `json:"error_old,omitempty"`
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

	if r.ErrorOld != nil {
		_, err = fmt.Fprintf(s.writer, "old error:\t%s\n", r.ErrorOld)
		if err != nil {
			panic(err)
		}
	}

	if r.Error != nil {
		_, err = fmt.Fprintf(s.writer, "new error:\t%s\n", r.Error)
		if err != nil {
			panic(err)
		}
	}

	if len(r.BenchOldStr) > 0 {
		_, err = fmt.Fprintf(s.writer, "old bench:\t%s\n", r.BenchOldStr)
		if err != nil {
			panic(err)
		}
	}

	if len(r.BenchStr) > 0 {
		_, err = fmt.Fprintf(s.writer, "new bench:\t%s\n", r.BenchStr)
		if err != nil {
			panic(err)
		}
	}

	if len(r.Diff) > 0 {
		_, err = fmt.Fprintf(s.writer, "mismatch:\n%s", r.Diff)
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

func run(paths []string, bench bool, old bool, compare bool, json bool) {
	if len(paths) == 0 {
		paths = []string{""}
	}

	var out output
	if json {
		out = newJSONOutput(len(paths))
	} else {
		out = newStdoutOutput()
	}

	allSucceeded := true

	for _, path := range paths {
		res, runSucceeded := runPath(path, old, compare, bench)
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

func runPath(path string, old bool, compare bool, bench bool) (res result, succeeded bool) {
	res = result{
		Path: path,
	}
	succeeded = true

	code := read(path)

	var newResult *ast.Program
	var newErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				newErr = fmt.Errorf("%s", debug.Stack())
				res.Error = newErr
			}
		}()

		newResult, newErr = parser2.ParseProgram(code)
		res.Error = newErr
	}()

	var oldResult *ast.Program
	var oldErr error
	if old || compare {
		func() {
			defer func() {
				if r := recover(); r != nil {
					oldErr = fmt.Errorf("%s", debug.Stack())
					res.ErrorOld = oldErr
				}
			}()

			oldResult, _, oldErr = parser1.ParseProgram(code)
			res.ErrorOld = oldErr
		}()
	}

	if newErr != nil || oldErr != nil {
		succeeded = false
	}

	if compare && newErr == nil && oldErr == nil {
		diff := compareOldAndNew(oldResult, newResult)
		if len(diff) > 0 {
			succeeded = false
		}
		res.Diff = diff
	}

	if bench {
		if newErr == nil {
			benchRes := benchParse(code, func(code string) (err error) {
				_, err = parser2.ParseProgram(code)
				return
			})
			res.Bench = &benchResult{
				Iterations: benchRes.N,
				Time:       benchRes.T,
			}
			res.BenchStr = benchRes.String()
		}

		if old && oldErr == nil {
			benchRes := benchParse(code, func(code string) (err error) {
				_, _, err = parser1.ParseProgram(code)
				return
			})
			res.BenchOld = &benchResult{
				Iterations: benchRes.N,
				Time:       benchRes.T,
			}
			res.BenchOldStr = benchRes.String()
		}
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

func benchParse(code string, parse func(string) error) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := parse(code)
			if err != nil {
				panic(err)
			}
		}
	})
}

func compareOldAndNew(oldResult, newResult *ast.Program) string {
	// the maximum levels of a struct to recurse into
	// this prevents infinite recursion from circular references
	deep.MaxDepth = 100

	diff := deep.Equal(oldResult, newResult)

	s := strings.Builder{}

	for _, d := range diff {
		s.WriteString(d)
		s.WriteRune('\n')
	}

	return s.String()
}
