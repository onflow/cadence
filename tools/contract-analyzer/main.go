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
	"sort"
	"sync"

	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-cli/pkg/flowkit/config"
	"github.com/onflow/flow-cli/pkg/flowkit/gateway"
	"github.com/onflow/flow-cli/pkg/flowkit/output"
	"github.com/onflow/flow-cli/pkg/flowkit/services"
	"github.com/onflow/flow-go-sdk"
	"github.com/spf13/afero"

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
func main() {
	var csvPathFlag = flag.String("csv", "", "analyze all programs in the given CSV file")
	var networkFlag = flag.String("network", "", "name of network")
	var addressFlag = flag.String("address", "", "analyze contracts in the given account")
	var loadOnlyFlag = flag.Bool("load-only", false, "only load (parse and check) programs")
	var analyzersFlag stringSliceFlag
	flag.Var(&analyzersFlag, "analyze", "enable analyzer")

	defaultUsage := flag.Usage
	flag.Usage = func() {
		defaultUsage()
		_, _ = fmt.Fprintf(os.Stderr, "\nAvailable analyzers:\n")

		names := make([]string, 0, len(analyzers.Analyzers))
		for name := range analyzers.Analyzers {
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			analyzer := analyzers.Analyzers[name]
			_, _ = fmt.Fprintf(
				os.Stderr,
				"  - %s:\n      %s\n",
				name,
				analyzer.Description,
			)
		}
	}

	flag.Parse()

	var enabledAnalyzers []*analysis.Analyzer

	loadOnly := *loadOnlyFlag
	if !loadOnly {
		if len(analyzersFlag) > 0 {
			for _, analyzerName := range analyzersFlag {
				analyzer, ok := analyzers.Analyzers[analyzerName]
				if !ok {
					log.Panic(fmt.Errorf("unknown analyzer: %s", analyzerName))
				}

				enabledAnalyzers = append(enabledAnalyzers, analyzer)
			}
		} else {
			for _, analyzer := range analyzers.Analyzers {
				enabledAnalyzers = append(enabledAnalyzers, analyzer)
			}
		}
	}

	cvsPath := *csvPathFlag
	address := *addressFlag
	switch {
	case cvsPath != "":
		analyzeCSV(cvsPath, enabledAnalyzers)

	case address != "":
		network := *networkFlag
		analyzeAccount(address, network, enabledAnalyzers)

	default:
		println("Nothing to do. Please provide -address or -csv. See -help")
	}
}

func analyzeAccount(address string, networkName string, analyzers []*analysis.Analyzer) {
	loader := &afero.Afero{Fs: afero.NewOsFs()}
	state, err := flowkit.Load(config.DefaultPaths(), loader)
	if err != nil {
		panic(err)
	}

	network, err := state.Networks().ByName(networkName)
	if err != nil {
		panic(err)
	}

	grpcGateway, err := gateway.NewGrpcGateway(network.Host)
	if err != nil {
		panic(err)
	}

	logger := output.NewStdoutLogger(output.ErrorLog)

	services := services.NewServices(grpcGateway, state, logger)

	codes := map[common.LocationID]string{}
	contractNames := map[common.Address][]string{}

	getContracts := func(flowAddress flow.Address) (map[string][]byte, error) {
		account, err := services.Accounts.Get(flowAddress)
		if err != nil {
			return nil, err
		}

		return account.Contracts, nil
	}

	flowAddress := flow.HexToAddress(address)
	commonAddress := common.Address(flowAddress)

	contracts, err := getContracts(flowAddress)
	if err != nil {
		panic(err)
	}

	locations := make([]common.Location, 0, len(contracts))
	for contractName := range contracts {
		location := common.AddressLocation{
			Address: commonAddress,
			Name:    contractName,
		}
		locations = append(locations, location)
	}

	analysisConfig := analysis.NewSimpleConfig(
		analysis.NeedTypes,
		codes,
		contractNames,
		func(address common.Address) (map[string]string, error) {
			contracts, err := getContracts(flow.Address(address))
			if err != nil {
				return nil, err
			}
			codes := make(map[string]string, len(contracts))
			for name, bytes := range contracts {
				codes[name] = string(bytes)
			}
			return codes, nil
		},
	)
	analyze(analysisConfig, locations, codes, analyzers)
}

func analyzeCSV(path string, analyzers []*analysis.Analyzer) {

	csvFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(csvFile)

	locations, codes, contractNames := readCSV(csvFile)
	analysisConfig := analysis.NewSimpleConfig(
		analysis.NeedTypes,
		codes,
		contractNames,
		nil,
	)
	analyze(analysisConfig, locations, codes, analyzers)
}

func analyze(
	config *analysis.Config,
	locations []common.Location,
	codes map[common.LocationID]string,
	analyzers []*analysis.Analyzer,
) {
	programs := make(analysis.Programs, len(locations))

	log.Println("Loading ...")

	for _, location := range locations {
		log.Printf("Loading %s", location)

		err := programs.Load(config, location)
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

	if len(analyzers) > 0 {
		for _, location := range locations {
			program := programs[location.ID()]
			if program == nil {
				continue
			}

			log.Printf("Analyzing %s", location)

			program.Run(analyzers, report)
		}
	}
}

func readCSV(
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
