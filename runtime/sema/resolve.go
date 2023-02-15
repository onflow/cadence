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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type AddressContractNamesResolver func(address common.Address) ([]string, error)

// AddressLocationHandlerFunc returns a location handler
// which returns a single location for non-address locations,
// and uses the given address contract names resolve function
// to get all contract names for an address
func AddressLocationHandlerFunc(resolveAddressContractNames AddressContractNamesResolver) LocationHandlerFunc {
	return func(identifiers []ast.Identifier, location common.Location) ([]ResolvedLocation, error) {
		addressLocation, isAddress := location.(common.AddressLocation)

		// if the location is not an address location, e.g. an identifier location (`import Crypto`),
		// then return a single resolved location which declares all identifiers.

		if !isAddress {
			return []ResolvedLocation{
				{
					Location:    location,
					Identifiers: identifiers,
				},
			}, nil
		}

		// if the location is an address,
		// and no specific identifiers where requested in the import statement,
		// then fetch all identifiers at this address

		if len(identifiers) == 0 {
			// if there is no contract name resolver,
			// then return no resolved locations

			if resolveAddressContractNames == nil {
				return nil, nil
			}

			contractNames, err := resolveAddressContractNames(addressLocation.Address)
			if err != nil {
				panic(err)
			}

			// if there are no contracts deployed,
			// then return no resolved locations

			if len(contractNames) == 0 {
				return nil, nil
			}

			identifiers = make([]ast.Identifier, len(contractNames))

			for i := range identifiers {
				identifiers[i] = ast.Identifier{
					Identifier: contractNames[i],
				}
			}
		}

		// return one resolved location per identifier.
		// each resolved location is an address contract location

		resolvedLocations := make([]ResolvedLocation, len(identifiers))
		for i, identifier := range identifiers {
			resolvedLocations[i] = ResolvedLocation{
				Location: common.AddressLocation{
					Address: addressLocation.Address,
					Name:    identifier.Identifier,
				},
				Identifiers: []ast.Identifier{
					identifier,
				},
			}
		}

		return resolvedLocations, nil
	}
}
