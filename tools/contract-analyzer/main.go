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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/tools/analysis"
)

var errorPrettyPrinter = pretty.NewErrorPrettyPrinter(os.Stdout, true)

func printErr(err error, location common.Location, codes map[common.LocationID]string) {
	printErr := errorPrettyPrinter.PrettyPrintError(err, location, codes)
	if printErr != nil {
		panic(printErr)
	}
}

func main() {
	var analyzersFlag stringSliceFlag
	flag.Var(&analyzersFlag, "a", "enable analyzer")
	flag.Parse()

	var enabledAnalyzers []analyzer

	for _, analyzerName := range analyzersFlag {
		analyzer, ok := analyzers[analyzerName]
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
	analysisConfig := newAnalysisConfig(codes, contractNames)

	programs := make(analysis.Programs, len(locations))

	log.Println("Loading ...")

	for _, location := range locations {
		log.Printf("Loading %s", location)

		err := programs.Load(analysisConfig, location)
		if err != nil {
			printErr(err, location, codes)
		}
	}

	report := func(err error, location common.Location) {
		printErr(err, location, codes)
	}

	for _, analyzer := range enabledAnalyzers {

		log.Printf("Runing analyzer %s ...", analyzer.Name())

		for _, location := range locations {
			log.Printf("Analyzing %s", location)

			program := programs[location.ID()]
			if program != nil {
				analyzer.Analyze(program, report)
			}
		}
	}
}

func newAnalysisConfig(
	codes map[common.LocationID]string,
	contractNames map[common.Address][]string,
) *analysis.Config {
	config := &analysis.Config{
		Mode: analysis.NeedTypes,
		ResolveAddressContractNames: func(
			address common.Address,
		) (
			[]string,
			error,
		) {
			names, ok := contractNames[address]
			if !ok {
				return nil, fmt.Errorf(
					"missing contracts for address: %s",
					address,
				)
			}
			return names, nil
		},
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) (
			string,
			error,
		) {
			code, ok := codes[location.ID()]
			if !ok {
				return "", fmt.Errorf(
					"import of unknown location: %s",
					location,
				)
			}

			return code, nil
		},
	}
	return config
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
