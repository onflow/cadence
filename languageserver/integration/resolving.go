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
	"fmt"
	"github.com/onflow/cadence"
	"io/ioutil"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go-sdk"
)

func resolveFileImport(location common.StringLocation) (string, error) {
	filename := string(location)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (i *FlowIntegration) resolveAddressContractNames(address common.Address) ([]string, error) {
	account, err := i.getAccount(address)
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

func (i *FlowIntegration) resolveAddressImport(location common.AddressLocation) (string, error) {

	account, err := i.getAccount(location.Address)
	if err != nil {
		return "", err
	}

	return string(account.Contracts[location.Name]), nil
}

func (i *FlowIntegration) getAccount(address common.Address) (*flow.Account, error) {
	account, err := i.sharedServices.Accounts.Get(
		flow.HexToAddress(address.String()),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"cannot get account with address %s: %w",
			address.ShortHexWithPrefix(),
			err,
		)
	}

	return account, nil
}

func (i *FlowIntegration) getAccountAddress(name string) (flow.Address, error) {
	serviceAccount, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return flow.Address{}, err
	}

	cadenceName, err := cadence.NewString(name)
	if err != nil {
		return flow.Address{}, err
	}
	args := []cadence.Value{cadenceName}

	code := makeManagerCode(scriptGetAddress, serviceAccount.Address().String())
	result, err := i.sharedServices.Scripts.Execute(code, args, "", "")
	if err != nil {
		return flow.Address{}, err
	}
	return flow.HexToAddress(result.String()), nil
}
