/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/hasura/go-graphql-client"
)

type chainID string

const (
	mainnet chainID = "mainnet"
	testnet chainID = "testnet"
)

var chainFlag = flag.String("chain", "", "mainnet or testnet")
var apiKeyFlag = flag.String("apiKey", "", "Flowdiver API key")
var batchFlag = flag.Int("batch", 500, "batch size")

var csvHeader = []string{"location", "code"}

func main() {
	flag.Parse()

	// Get batch size from flags

	batchSize := *batchFlag

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

	// Get API key from flags

	apiKey := *apiKeyFlag
	if apiKey == "" {
		log.Fatal("missing Flowdiver API key")
	}

	// Get contracts from network

	var apiURL string
	switch chain {
	case mainnet:
		apiURL = "https://api.findlabs.io/hasura/v1/graphql"
	case testnet:
		apiURL = "https://api.findlabs.io/hasura_testnet/v1/graphql"
	}

	client := graphql.NewClient(apiURL, nil).
		WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			// NOTE: important, default is forbidden
			r.Header.Set("User-Agent", "")
		})

	var total, offset int
	var contracts [][]string

	for {

		log.Printf("fetching contracts %d-%d", offset, offset+batchSize)

		var req struct {
			ContractsAggregate struct {
				Aggregate struct {
					Count int
				}
			} `graphql:"contracts_aggregate(where: {valid_to: {_is_null: true}})"`
			Contracts []struct {
				Identifier string
				Body       string
			} `graphql:"contracts(where: {valid_to: {_is_null: true}}, limit: $limit, offset: $offset)"`
		}

		if err := client.Query(
			context.Background(),
			&req,
			map[string]any{
				"offset": offset,
				"limit":  batchSize,
			},
		); err != nil {
			log.Fatalf("failed to query: %s", err)
		}

		total = req.ContractsAggregate.Aggregate.Count

		if contracts == nil {
			contracts = make([][]string, 0, total)
		}

		for _, contract := range req.Contracts {
			contracts = append(
				contracts, []string{
					contract.Identifier,
					contract.Body,
				},
			)
		}

		offset += batchSize

		if offset >= total {
			break
		}
	}

	// Sort

	sort.Slice(
		contracts,
		func(i, j int) bool {
			return contracts[i][0] < contracts[j][0]
		},
	)

	// Write contracts to CSV

	writer := csv.NewWriter(os.Stdout)

	if err := writer.Write(csvHeader); err != nil {
		log.Fatalf("failed to write CSV header: %s", err)
		return
	}

	for _, contract := range contracts {
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
