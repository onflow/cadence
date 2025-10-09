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
	"os"

	"github.com/rs/zerolog/log"

	"github.com/onflow/cadence/tools/compatibility_check"

	"github.com/onflow/flow-go/model/flow"
)

func main() {
	if len(os.Args) < 3 {
		log.Error().Msg("not enough arguments. Usage: <chain_name> <csv_path> <output_path>")
		return
	}

	chainName := os.Args[1]
	csvPath := os.Args[2]
	outputPath := os.Args[3]

	chain := flow.ChainID(chainName).Chain()

	csvFile, err := os.Open(csvPath)
	if err != nil {
		log.Err(err).Msgf("failed to open csv file: %s", csvPath)
		return
	}
	defer func() {
		_ = csvFile.Close()
	}()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Err(err).Msgf("failed to create output file: %s", outputPath)
		return
	}
	defer func() {
		_ = outputFile.Close()
	}()

	checker := compatibility_check.NewContractChecker(chain, outputFile)
	checker.CheckCSV(csvFile)
}
