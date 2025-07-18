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

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/cmd"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/pretty"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

type memberAccountAccessFlags []string

func (f *memberAccountAccessFlags) String() string {
	return ""
}

func (f *memberAccountAccessFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

var benchFlag = flag.Bool("bench", false, "benchmark the checker")
var jsonFlag = flag.Bool("json", false, "print the result formatted as JSON")

var memberAccountAccessFlag memberAccountAccessFlags

func main() {
	flag.Var(&memberAccountAccessFlag, "memberAccountAccess", "allow account access from:to")
	flag.Parse()

	memberAccountAccess := map[common.Location]map[common.Location]struct{}{}

	for _, value := range memberAccountAccessFlag {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) < 2 {
			panic(fmt.Errorf("invalid member access flag: got '%s', expected 'from:to'", value))
		}

		source := parts[0]
		sourceLocation, _, err := common.DecodeTypeID(nil, source)
		if err != nil {
			panic(fmt.Errorf("invalid member access source location: %s: %w", source, err))
		}

		target := parts[1]
		targetLocation, _, err := common.DecodeTypeID(nil, target)
		if err != nil {
			panic(fmt.Errorf("invalid member access target location: %s: %w", target, err))
		}

		nested := memberAccountAccess[sourceLocation]
		if nested == nil {
			nested = map[common.Location]struct{}{}
			memberAccountAccess[sourceLocation] = nested
		}
		nested[targetLocation] = struct{}{}
	}

	args := flag.Args()
	run(args, *benchFlag, *jsonFlag, memberAccountAccess)
}

type benchResult struct {
	// N is the number of iterations
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

func run(
	paths []string,
	bench bool,
	json bool,
	memberAccountAccess map[common.Location]map[common.Location]struct{},
) {
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
		res, runSucceeded := runPath(path, bench, useColor, memberAccountAccess)
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

func runPath(
	path string,
	bench bool,
	useColor bool,
	memberAccountAccess map[common.Location]map[common.Location]struct{},
) (res result, succeeded bool) {
	res = result{
		Path: path,
	}
	succeeded = true

	code := read(path)

	var err error
	var checker *sema.Checker
	var program *ast.Program
	var must func(error)

	codes := map[common.Location][]byte{}

	location := common.NewStringLocation(nil, path)

	// standard library handler is only needed for execution, but we're only checking
	standardLibraryValues := stdlib.InterpreterDefaultScriptStandardLibraryValues(nil)

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%s", debug.Stack())
				res.Error = err.Error()
			}
		}()

		program, must = cmd.PrepareProgram(code, location, codes)

		checker, _ = cmd.PrepareChecker(
			program,
			location,
			codes,
			memberAccountAccess,
			standardLibraryValues,
			must,
		)

		err = checker.Check()
		if err != nil {
			var builder strings.Builder
			printErr := pretty.NewErrorPrettyPrinter(&builder, useColor).
				PrettyPrintError(err, location, codes)
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

				checker, must = cmd.PrepareChecker(
					program,
					location,
					codes,
					memberAccountAccess,
					standardLibraryValues,
					must,
				)
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

func read(path string) []byte {
	var data []byte
	var err error
	if len(path) == 0 {
		data, err = io.ReadAll(bufio.NewReader(os.Stdin))
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		panic(err)
	}
	return data
}
