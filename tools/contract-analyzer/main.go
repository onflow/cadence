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
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/tools/analysis"
	"github.com/onflow/cadence/tools/contract-analyzer/analyzers"
)

var errorPrettyPrinter = pretty.NewErrorPrettyPrinter(os.Stdout, true)

func printErr(err error, location common.Location, codes map[common.LocationID]string) {
	printErr := errorPrettyPrinter.PrettyPrintError(err, location, codes)
	if printErr != nil {
		panic(printErr)
	}
}

type diagnosticErr struct {
	analysis.Diagnostic
}

var _ error = diagnosticErr{}

func (d diagnosticErr) Error() string {
	return d.Message
}

func main() {
	var analyzersFlag stringSliceFlag
	flag.Var(&analyzersFlag, "a", "enable analyzer")

	defaultUsage := flag.Usage
	flag.Usage = func() {
		defaultUsage()
		_, _ = fmt.Fprintf(os.Stderr, "\nAvailable analyzers:\n")
		for name := range analyzers.Analyzers {
			_, _ = fmt.Fprintf(os.Stderr, "  - %s\n", name)
		}
	}

	flag.Parse()

	var enabledAnalyzers []*analysis.Analyzer

	for _, analyzerName := range analyzersFlag {
		analyzer, ok := analyzers.Analyzers[analyzerName]
		if !ok {
			log.Panic(fmt.Errorf("unknown analyzer: %s", analyzerName))
		}

		enabledAnalyzers = append(enabledAnalyzers, analyzer)
	}

	var file *os.File
	if flag.NArg() > 0 {
		var err error
		file, err = os.Open(flag.Arg(0))
		if err != nil {
			panic(err)
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(file)
	} else {
		file = os.Stdin
	}

	locations, codes, contractNames := readContracts(file)
	analysisConfig := analysis.NewSimpleConfig(analysis.NeedTypes, codes, contractNames)

	programs := make(analysis.Programs, len(locations))

	log.Println("Loading ...")

	for _, location := range locations {
		log.Printf("Loading %s", location)

		err := programs.Load(analysisConfig, location)
		if err != nil {
			printErr(err, location, codes)
		}
	}

	var reportLock sync.Mutex

	report := func(diagnostic analysis.Diagnostic) {
		reportLock.Lock()
		defer reportLock.Unlock()

		printErr(
			diagnosticErr{diagnostic},
			diagnostic.Location,
			codes,
		)
	}

	for _, location := range locations {
		program := programs[location.ID()]
		if program == nil {
			continue
		}

		log.Printf("Analyzing %s", location)

		program.Run(enabledAnalyzers, report)
	}
}

func readContracts(
	r io.Reader,
) (
	locations []common.Location,
	codes map[common.LocationID]string,
	contractNames map[common.Address][]string,
) {
	reader := csv.NewReader(r)

	codes = map[common.LocationID]string{}
	contractNames = map[common.Address][]string{}

	var record []string
	for {
		var err error
		skip := record == nil
		record, err = reader.Read()
		if err == io.EOF {
			break
		}
		if skip {
			continue
		}

		address, _ := common.HexToAddress(record[0])
		name := record[1]
		code := record[2]

		location := common.AddressLocation{
			Address: address,
			Name:    name,
		}

		locations = append(locations, location)
		codes[location.ID()] = code
		contractNames[address] = append(contractNames[address], name)
	}

	return
}
