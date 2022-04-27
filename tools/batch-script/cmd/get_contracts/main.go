/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

// Get all contracts from a network and write them as a CSV file to standard output.
// The CSV file has the header: address,name,code
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"os"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/tools/batch-script"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var url = flag.String("u", "", "Flow Access Node URL")

var csvHeader = []string{"address", "name", "code"}

func main() {
	flag.Parse()

	config := batch_script.DefaultConfig
	if *url != "" {
		config.FlowAccessNodeURL = *url
	}

	log.Logger = log.
		Output(zerolog.ConsoleWriter{Out: os.Stderr}).
		Level(zerolog.InfoLevel)

	contracts := make(chan []string)

	go func() {
		err := batch_script.BatchScript(
			context.Background(),
			log.Logger,
			config,
			batch_script.GetContracts,
			batch_script.NewGetContractsHandler(
				func(address cadence.Address, contractName, contractCode string, err error) {
					if err != nil {
						log.Err(err).Msg("failed to get contract info")
						return
					}
					contracts <- []string{
						address.Hex(),
						contractName,
						contractCode,
					}
				},
			),
		)

		close(contracts)

		if err != nil {
			log.Err(err).Msg("batch script failed")
			return
		}
	}()

	writer := csv.NewWriter(os.Stdout)

	if err := writer.Write(csvHeader); err != nil {
		log.Err(err).Msg("failed to write CSV header")
		return
	}

	for contract := range contracts {
		err := writer.Write(contract)
		if err != nil {
			log.Err(err).Msg("failed to write contract to CSV")
			return
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		log.Err(err).Msg("failed to write CSV")
	}
}
