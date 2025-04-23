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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type contractLoader interface {
	interpreter.StorageMutationTracker
	common.MemoryGauge
}

func loadContractValue(
	loader contractLoader,
	location common.AddressLocation,
	storage *Storage,
) *interpreter.CompositeValue {

	var contractValue interpreter.Value

	storageMap := storage.GetDomainStorageMap(
		loader,
		location.Address,
		common.StorageDomainContract,
		false,
	)
	if storageMap != nil {
		contractValue = storageMap.ReadValue(
			loader,
			interpreter.StringStorageMapKey(location.Name),
		)
	}

	if contractValue == nil {
		panic(errors.NewDefaultUserError("failed to load contract: %s", location))
	}

	return contractValue.(*interpreter.CompositeValue)
}
