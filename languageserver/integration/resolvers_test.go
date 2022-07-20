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
	"testing"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go-sdk"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_FileImport(t *testing.T) {
	t.Parallel()
	mockFS := afero.NewMemMapFs()
	af := afero.Afero{Fs: mockFS}
	_ = afero.WriteFile(mockFS, "./test.cdc", []byte(`hello test`), 0644)

	resolver := resolvers{
		loader: af,
	}

	t.Run("existing file", func(t *testing.T) {
		resolved, err := resolver.fileImport("./test.cdc")
		assert.NoError(t, err)
		assert.Equal(t, "hello test", resolved)
	})

	t.Run("non existing file", func(t *testing.T) {
		resolved, err := resolver.fileImport("./foo.cdc")
		assert.EqualError(t, err, "open foo.cdc: file does not exist")
		assert.Equal(t, "", resolved)
	})
}

func Test_AddressImport(t *testing.T) {
	mock := &mockFlowClient{}
	resolver := resolvers{
		client: mock,
	}

	a, _ := common.HexToAddress("1")
	address := common.NewAddressLocation(nil, a, "test")
	flowAddress := flow.HexToAddress(a.String())

	mock.
		On("GetAccount", flowAddress).
		Return(&flow.Account{
			Address: flowAddress,
			Contracts: map[string][]byte{
				"test": []byte("hello tests"),
				"foo":  []byte("foo bar"),
			},
		}, nil)

	nonExisting := flow.HexToAddress("2")
	mock.
		On("GetAccount", nonExisting).
		Return(nil, fmt.Errorf("failed to get account with address %s", nonExisting.String()))

	t.Run("existing address", func(t *testing.T) {
		resolved, err := resolver.addressImport(address)
		assert.NoError(t, err)
		assert.Equal(t, "hello tests", resolved)
	})

	t.Run("non existing contract import", func(t *testing.T) {
		address.Name = "invalid"
		resolved, err := resolver.addressImport(address)
		assert.NoError(t, err)
		assert.Empty(t, resolved)
	})

	t.Run("non existing address", func(t *testing.T) {
		address.Address, _ = common.HexToAddress("2")
		resolved, err := resolver.addressImport(address)
		assert.EqualError(t, err, "failed to get account with address 0000000000000002")
		assert.Empty(t, resolved)
	})

	t.Run("address contract names", func(t *testing.T) {
		contracts, err := resolver.addressContractNames(a)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"foo", "test"}, contracts)
	})
}
