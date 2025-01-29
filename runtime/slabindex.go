/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package runtime

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

// readSlabIndexFromRegister returns register value as atree.SlabIndex.
// This function returns error if
// - underlying ledger panics, or
// - underlying ledger returns error when retrieving ledger value, or
// - retrieved ledger value is invalid (for atree.SlabIndex).
func readSlabIndexFromRegister(
	ledger atree.Ledger,
	address common.Address,
	key []byte,
) (atree.SlabIndex, bool, error) {
	var data []byte
	var err error
	errors.WrapPanic(func() {
		data, err = ledger.GetValue(address[:], key)
	})
	if err != nil {
		return atree.SlabIndex{}, false, interpreter.WrappedExternalError(err)
	}

	dataLength := len(data)

	if dataLength == 0 {
		return atree.SlabIndex{}, false, nil
	}

	isStorageIndex := dataLength == storageIndexLength
	if !isStorageIndex {
		// Invalid data in register

		// TODO: add dedicated error type?
		return atree.SlabIndex{}, false, errors.NewUnexpectedError(
			"invalid storage index for storage map of account '%x': expected length %d, got %d",
			address[:], storageIndexLength, dataLength,
		)
	}

	return atree.SlabIndex(data), true, nil
}

func writeSlabIndexToRegister(
	ledger atree.Ledger,
	address common.Address,
	key []byte,
	slabIndex atree.SlabIndex,
) error {
	var err error
	errors.WrapPanic(func() {
		err = ledger.SetValue(
			address[:],
			key,
			slabIndex[:],
		)
	})
	if err != nil {
		return interpreter.WrappedExternalError(err)
	}
	return nil
}
