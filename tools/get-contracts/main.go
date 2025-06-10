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

// Get all contracts from a network and write them as a CSV file to standard output.
// The CSV file has the header: location,code
package main

import (
	"encoding/base64"
	"encoding/csv"
	"flag"
	"log"
	"net/http"
	"os"
	"sort"
)

type chainID string

const (
	mainnet chainID = "mainnet"
	testnet chainID = "testnet"
)

var chainFlag = flag.String("chain", "", "mainnet or testnet")

const authFlagUsage = "find.xyz API auth (username:password)"

var authFlag = flag.String("auth", "", authFlagUsage)

var resultCSVHeader = []string{"location", "code"}

func main() {
	flag.Parse()

	// Get chain ID from flags

	chain := chainID(*chainFlag)
	switch chain {
	case mainnet, testnet:
		break
	case "":
		log.Fatal("missing chain ID")
	default:
		log.Fatalf("invalid chain: %s", chain)
	}

	// Get auth from flags

	auth := *authFlag
	if auth == "" {
		log.Fatal("missing " + authFlagUsage)
	}

	// Get contracts from network

	var apiURL string
	switch chain {
	case mainnet:
		apiURL = "https://api.find.xyz"
	case testnet:
		apiURL = "https://api.test-find.xyz"
	}

	apiURL += "/bulk/v1/contract?valid_only=true"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Fatalf("failed to create HTTP request: %s", err)
	}

	req.Header.Set("Accept", "text/csv")
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("failed to send HTTP request: %s", err)
	}

	if res.StatusCode != http.StatusOK {
		log.Fatalf("unexpected status code: %d", res.StatusCode)
	}

	reader := csv.NewReader(res.Body)
	reader.FieldsPerRecord = -1

	contracts, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("failed to read CSV: %s", err)
	}

	// Skip header
	contracts = contracts[1:]

	// Sort

	sort.Slice(
		contracts,
		func(i, j int) bool {
			return contracts[i][0] < contracts[j][0]
		},
	)

	// Write contracts to CSV

	writer := csv.NewWriter(os.Stdout)

	if err := writer.Write(resultCSVHeader); err != nil {
		log.Fatalf("failed to write CSV header: %s", err)
		return
	}

	for _, contract := range contracts {
		identifier := contract[0]
		if identifier == "A." || identifier == "null" {
			continue
		}

		err := writer.Write(contract)
		if err != nil {
			log.Fatalf("failed to write contract to CSV: %s", err)
			return
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		log.Fatalf("failed to write CSV: %s", err)
	}
}
