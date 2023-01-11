/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2023 Dapper Labs, Inc.
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

package compatibility_check

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"encoding/csv"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/tools/analysis"
)

const LoadMode = analysis.NeedTypes

type ContractsChecker struct {
	Codes              map[common.Location][]byte
	errorPrettyPrinter pretty.ErrorPrettyPrinter
}

func NewContractChecker(outputFile *os.File) *ContractsChecker {
	checker := &ContractsChecker{
		Codes:              map[common.Location][]byte{},
		errorPrettyPrinter: pretty.NewErrorPrettyPrinter(outputFile, false),
	}

	return checker
}

func (c *ContractsChecker) CheckCSV(csvFile *os.File) {
	locations, contractNames := c.readCSV(csvFile)
	analysisConfig := analysis.NewSimpleConfig(
		LoadMode,
		c.Codes,
		contractNames,
		nil,
	)

	c.analyze(analysisConfig, locations)
}

func (c *ContractsChecker) readCSV(
	r io.Reader,
) (
	locations []common.Location,
	contractNames map[common.Address][]string,
) {
	reader := csv.NewReader(r)

	contractNames = map[common.Address][]string{}

	var record []string
	for rowNumber := 1; ; rowNumber++ {
		skip := record == nil
		var err error
		record, err = reader.Read()
		if err == io.EOF {
			break
		}
		if skip {
			continue
		}

		location, qualifiedIdentifier, err := common.DecodeTypeID(nil, record[0])
		if err != nil {
			panic(fmt.Errorf("invalid location in row %d: %w", rowNumber, err))
		}

		identifierParts := strings.Split(qualifiedIdentifier, ".")
		if len(identifierParts) > 1 {
			panic(fmt.Errorf(
				"invalid location in row %d: invalid qualified identifier: %s",
				rowNumber,
				qualifiedIdentifier,
			))
		}

		code := record[1]
		locations = append(locations, location)
		c.Codes[location] = []byte(code)

		if addressLocation, ok := location.(common.AddressLocation); ok {
			contractNames[addressLocation.Address] = append(
				contractNames[addressLocation.Address],
				addressLocation.Name,
			)
		}
	}

	return
}

func (c *ContractsChecker) analyze(
	config *analysis.Config,
	locations []common.Location,
) {
	programs := make(analysis.Programs, len(locations))

	log.Println("Checking contracts ...")

	for _, location := range locations {
		log.Printf("Checking %s", location.Description())

		err := programs.Load(config, location)
		if err != nil {
			c.prettyPrintError(err, location)
		}
	}
}

func (c *ContractsChecker) prettyPrintError(err error, location common.Location) {
	printErr := c.errorPrettyPrinter.PrettyPrintError(err, location, c.Codes)
	if printErr != nil {
		panic(printErr)
	}
}
