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
// The CSV file has the header: location,code
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"os"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/onflow/cadence/tools/batch-script"
)

var urlFlag = flag.String("u", "", "Flow Access Node URL")
var pauseFlag = flag.String("p", "", "pause duration")

var csvHeader = []string{"location", "code"}

func main() {
	flag.Parse()

	config := batch_script.DefaultConfig

	url := *urlFlag
	if url != "" {
		config.FlowAccessNodeURL = url
	}

	pause, err := time.ParseDuration(*pauseFlag)
	if err == nil {
		config.Pause = pause
	} else {
		log.Error().Msg("invalid pause duration")
		return
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
					location := common.AddressLocation{
						Address: common.Address(address),
						Name:    contractName,
					}
					contracts <- []string{
						string(location.ID()),
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
