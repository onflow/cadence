//go:build !wasm
// +build !wasm

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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/itchyny/gojq"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"
)

var benchFlag = flag.Bool("bench", false, "benchmark the parser")
var jsonFlag = flag.Bool("json", false, "print the result formatted as JSON")
var readCSVFlag = flag.Bool("readCSV", false, "read the input file as CSV (header: location,code)")
var jqASTFlag = flag.String("jqAST", "", "query the AST using gojq")

func main() {
	testing.Init()
	flag.Parse()

	args := flag.Args()
	run(args, *benchFlag, *jsonFlag, *readCSVFlag, *jqASTFlag)
}

type benchResult struct {
	// N is the number of iterations
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
	Results  []any        `json:"results,omitempty"`
}

type output interface {
	Append(result)
	End()
}

type jsonOutput struct {
	file    *os.File
	results []result
}

func newJSONOutput(file *os.File, count int) *jsonOutput {
	return &jsonOutput{
		file:    file,
		results: make([]result, 0, count),
	}
}

func (j *jsonOutput) Append(r result) {
	j.results = append(j.results, r)
}

func (j *jsonOutput) End() {
	encoder := json.NewEncoder(j.file)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(j.results)
	if err != nil {
		panic(err)
	}
}

type fileOutput struct {
	file *os.File
}

func (s fileOutput) Append(r result) {

	if len(r.Path) > 0 {
		_, _ = fmt.Fprintf(s.file, "%s\n", r.Path)
	}

	if r.Error != nil {
		location := common.NewStringLocation(nil, r.Path)
		printErr := pretty.NewErrorPrettyPrinter(s.file, true).
			PrettyPrintError(r.Error, location, map[common.Location][]byte{location: r.Code})
		if printErr != nil {
			panic(printErr)
		}
	}

	if len(r.BenchStr) > 0 {
		_, _ = fmt.Fprintf(s.file, "bench:\t%s\n", r.BenchStr)
	}

	if len(r.Results) > 0 {
		_, _ = fmt.Fprint(s.file, "query results:\n")

		for _, res := range r.Results {
			_, _ = fmt.Fprintf(s.file, "- %#+v\n", res)
		}
	}

	println()
}

func queryProgram(program *ast.Program, query *gojq.Code) []any {
	// Encode to JSON
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(program)
	if err != nil {
		panic(err)
	}

	// Decode from JSON
	var decoded any
	err = json.NewDecoder(&buf).Decode(&decoded)
	if err != nil {
		panic(err)
	}

	var results []any

	// Run query and print results
	iter := query.Run(decoded)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			panic(err)
		}

		results = append(results, v)
	}

	return results
}

func (s fileOutput) End() {
	// no-op
}

func run(paths []string, bench bool, json bool, readCSV bool, jqAST string) {
	if len(paths) == 0 {
		paths = []string{""}
	}

	var compiledQuery *gojq.Code

	if jqAST != "" {
		query, err := gojq.Parse(jqAST)
		if err != nil {
			panic(err)
		}

		compiledQuery, err = gojq.Compile(query)
		if err != nil {
			panic(err)
		}

	}

	var out output
	if json {
		out = newJSONOutput(os.Stdout, len(paths))
	} else {
		out = fileOutput{file: os.Stdout}
	}

	allSucceeded := true

	for _, path := range paths {
		for _, file := range read(path, readCSV) {
			res, runSucceeded := runFile(file, bench, compiledQuery)
			if !runSucceeded {
				allSucceeded = false
			}
			out.Append(res)
		}
	}

	out.End()

	if !allSucceeded {
		os.Exit(1)
	}
}

func runFile(file file, bench bool, query *gojq.Code) (res result, succeeded bool) {
	res = result{
		Path: file.path,
	}
	succeeded = true

	code := file.code
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

		_, _ = fmt.Fprintf(os.Stderr, "parsing %s\n", file.path)

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

	if program != nil && query != nil {
		res.Results = queryProgram(program, query)
	}

	return
}

type file struct {
	path string
	code []byte
}

func read(path string, readCSV bool) []file {
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

	if readCSV {
		records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
		if err != nil {
			panic(err)
		}

		files := make([]file, 0, len(records))

		// Convert all records to files, except for the header
		for _, record := range records[1:] {
			files = append(files,
				file{
					path: record[0],
					code: []byte(record[1]),
				},
			)
		}

		return files
	} else {
		return []file{
			{
				path: path,
				code: data,
			},
		}
	}
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
