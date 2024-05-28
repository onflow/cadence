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

	"github.com/onflow/flow-go/cmd/util/ledger/util/registers"

	"github.com/onflow/cadence/runtime/common"
)

func addressesJSON(registersByAccount *registers.ByAccount) ([]byte, error) {

	var addresses []string

	err := registersByAccount.ForEachAccount(func(accountRegisters *registers.AccountRegisters) error {
		owner := accountRegisters.Owner()
		if len(owner) == 0 {
			return nil
		}

		address := common.Address([]byte(owner)).Hex()
		addresses = append(addresses, address)

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(addresses)

	encoded, err := json.Marshal(addresses)
	if err != nil {
		return nil, err
	}

	return encoded, nil
}
