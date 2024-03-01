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

package main

import (
	"encoding/json"
	"sort"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go/cmd/util/ledger/util"
)

func addressesJSON(payloadSnapshot *util.PayloadSnapshot) ([]byte, error) {
	addressSet := map[string]struct{}{}
	for registerID := range payloadSnapshot.Payloads {
		owner := registerID.Owner
		if len(owner) > 0 {
			address := common.Address([]byte(owner)).Hex()
			addressSet[address] = struct{}{}
		}
	}

	addresses := make([]string, 0, len(addressSet))
	for address := range addressSet {
		addresses = append(addresses, address)
	}

	sort.Strings(addresses)

	encoded, err := json.Marshal(addresses)
	if err != nil {
		return nil, err
	}

	return encoded, nil
}
