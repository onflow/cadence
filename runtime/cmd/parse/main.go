//go:build !wasm
// +build !wasm

/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"testing"
	"time"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"
)

var benchFlag = flag.Bool("bench", false, "benchmark the parser")
var jsonFlag = flag.Bool("json", false, "print the result formatted as JSON")

func main() {
	testing.Init()
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
	Error    error        `json:"error,omitempty"`
	Bench    *benchResult `json:"bench,omitempty"`
	Program  *ast.Program `json:"program"`
	Path     string       `json:"path,omitempty"`
	BenchStr string       `json:"-"`
	Code     []byte       `json:"-"`
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

type stdoutOutput struct{}

func (s stdoutOutput) Append(r result) {
	var err error

	if len(r.Path) > 0 {
		_, err = fmt.Printf("%s\n", r.Path)
		if err != nil {
			panic(err)
		}
	}

	if r.Error != nil {
		location := common.NewStringLocation(nil, r.Path)
		printErr := pretty.NewErrorPrettyPrinter(os.Stdout, true).
			PrettyPrintError(r.Error, location, map[common.Location][]byte{location: r.Code})
		if printErr != nil {
			panic(printErr)
		}
	}

	if len(r.BenchStr) > 0 {
		_, err = fmt.Printf("bench:\t%s\n", r.BenchStr)
		if err != nil {
			panic(err)
		}
	}
}

func (s stdoutOutput) End() {
	// no-op
}

func run(paths []string, bench bool, json bool) {
	if len(paths) == 0 {
		paths = []string{""}
	}

	var out output
	if json {
		out = newJSONOutput(len(paths))
	} else {
		out = stdoutOutput{}
	}

	allSucceeded := true

	for _, path := range paths {
		res, runSucceeded := runPath(path, bench)
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

func runPath(path string, bench bool) (res result, succeeded bool) {
	res = result{
		Path: path,
	}
	succeeded = true

	code := read(path)
	res.Code = code

	var program *ast.Program
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%s", debug.Stack())
				res.Error = err
			}
		}()

		program, err = parser.ParseProgram(nil, code, parser.Config{})
		if !bench {
			res.Program = program
		}
		res.Error = err
	}()

	if err != nil {
		succeeded = false
		return
	}

	if bench {
		benchRes := benchParse(func() (err error) {
			_, err = parser.ParseProgram(nil, code, parser.Config{})
			return
		})
		res.Bench = &benchResult{
			Iterations: benchRes.N,
			Time:       benchRes.T,
		}
		res.BenchStr = benchRes.String()
	}

	return
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

func benchParse(parse func() (err error)) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := parse()
			if err != nil {
				panic(err)
			}
		}
	})
}
