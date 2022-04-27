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

package batch_script

import (
	_ "embed"
	"encoding/hex"

	"github.com/onflow/cadence"
)

//go:embed get_contracts.cdc
var GetContracts string

type GetContractsHandler func(address cadence.Address, contractName, contractCode string, err error)

func NewGetContractsHandler(handler GetContractsHandler) func(value cadence.Value) {
	return func(value cadence.Value) {
		for _, addressContractsPair := range value.(cadence.Dictionary).Pairs {
			address := addressContractsPair.Key.(cadence.Address)
			for _, nameCodePair := range addressContractsPair.Value.(cadence.Dictionary).Pairs {
				name := string(nameCodePair.Key.(cadence.String))
				rawCode, err := hex.DecodeString(string(nameCodePair.Value.(cadence.String)))
				if err != nil {
					handler(cadence.Address{}, "", "", err)
					continue
				}
				code := string(rawCode)
				handler(address, name, code, nil)
			}
		}
	}
}
