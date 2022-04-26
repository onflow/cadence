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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// A Config specifies details about how programs should be loaded.
// The zero value is a valid configuration.
// Calls to Load do not modify this struct.
type Config struct {
	// Mode controls the level of information returned for each program.
	Mode LoadMode

	// ResolveAddressContractNames is called to resolve the contract names of an address location.
	ResolveAddressContractNames func(address common.Address) ([]string, error)

	// ResolveCode is called to resolve an import to its source code.
	ResolveCode func(
		location common.Location,
		importingLocation common.Location,
		importRange ast.Range,
	) (string, error)
}

func NewSimpleConfig(
	mode LoadMode,
	codes map[common.LocationID]string,
	contractNames map[common.Address][]string,
) *Config {
	config := &Config{
		Mode: mode,
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
