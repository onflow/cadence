/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go-sdk"
)

func resolveFileImport(mainPath string, location common.StringLocation) (string, error) {
	filename := path.Join(path.Dir(mainPath), string(location))

	if filename == mainPath {
		return "", fmt.Errorf("cannot import current file: %s", filename)
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (i *FlowIntegration) resolveAccountImport(location common.AddressLocation) (string, error) {
	accountAddr := location.ToAddress()

	acct, err := i.flowClient.GetAccount(context.Background(), flow.BytesToAddress(accountAddr[:]))
	if err != nil {
		return "", fmt.Errorf("cannot get account with address 0x%s. err: %w", accountAddr, err)
	}

	return string(acct.Code), nil
}
