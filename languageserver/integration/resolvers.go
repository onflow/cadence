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

package integration

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-go-sdk"
)

type resolvers struct {
	client flowClient
	loader flowkit.ReaderWriter
}

func (r *resolvers) fileImport(location common.StringLocation) (string, error) {
	filename := string(location)

	data, err := r.loader.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (r *resolvers) addressImport(location common.AddressLocation) (string, error) {
	account, err := r.client.GetAccount(flow.HexToAddress(location.Address.String()))
	if err != nil {
		return "", err
	}

	return string(account.Contracts[location.Name]), nil
}

func (r *resolvers) addressContractNames(address common.Address) ([]string, error) {
	account, err := r.client.GetAccount(flow.HexToAddress(address.String()))
	if err != nil {
		return nil, err
	}

	names := make([]string, len(account.Contracts))
	var index int
	for name := range account.Contracts {
		names[index] = name
		index++
	}

	return names, nil
}
