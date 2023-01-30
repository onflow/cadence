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

package analysis

import (
	"fmt"
	"sort"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// A Config specifies details about how programs should be loaded.
// The zero value is a valid configuration.
// Calls to Load do not modify this struct.
type Config struct {
	// ResolveAddressContractNames is called to resolve the contract names of an address location
	ResolveAddressContractNames func(address common.Address) ([]string, error)
	// ResolveCode is called to resolve an import to its source code
	ResolveCode func(
		location common.Location,
		importingLocation common.Location,
		importRange ast.Range,
	) ([]byte, error)
	// Mode controls the level of information returned for each program
	Mode LoadMode
}

func NewSimpleConfig(
	mode LoadMode,
	codes map[common.Location][]byte,
	contractNames map[common.Address][]string,
	resolveAddressContracts func(common.Address) (contracts map[string][]byte, err error),
) *Config {

	loadAddressContracts := func(address common.Address) error {
		if resolveAddressContracts == nil {
			return nil
		}
		contracts, err := resolveAddressContracts(address)
		if err != nil {
			return err
		}

		names := make([]string, 0, len(contracts))

		for name := range contracts { //nolint:maprange
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			code := contracts[name]
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			codes[location] = code
		}

		contractNames[address] = names

		return nil
	}

	config := &Config{
		Mode: mode,
		ResolveAddressContractNames: func(
			address common.Address,
		) (
			[]string,
			error,
		) {
			repeat := true
			for {
				names, ok := contractNames[address]
				if !ok {
					if repeat {
						err := loadAddressContracts(address)
						if err != nil {
							return nil, err
						}
						repeat = false
						continue
					}

					return nil, fmt.Errorf(
						"missing contracts for address: %s",
						address,
					)
				}
				return names, nil
			}
		},
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) (
			[]byte,
			error,
		) {
			repeat := true
			for {
				code, ok := codes[location]
				if !ok {
					if repeat {
						if addressLocation, ok := location.(common.AddressLocation); ok {
							err := loadAddressContracts(addressLocation.Address)
							if err != nil {
								return nil, err
							}
							repeat = false
							continue
						}
					}

					return nil, fmt.Errorf(
						"import of unknown location: %s",
						location,
					)
				}

				return code, nil
			}
		},
	}
	return config
}
