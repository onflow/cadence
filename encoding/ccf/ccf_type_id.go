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

package ccf

import (
	"fmt"
	"math/big"

	"github.com/onflow/cadence"
)

// ccfTypeID represents CCF type ID.
type ccfTypeID uint64

func newCCFTypeID(b []byte) ccfTypeID {
	return ccfTypeID(new(big.Int).SetBytes(b).Uint64())
}

func newCCFTypeIDFromUint64(i uint64) ccfTypeID {
	return ccfTypeID(i)
}

func (id ccfTypeID) Bytes() []byte {
	return new(big.Int).SetUint64(uint64(id)).Bytes()
}

func (id ccfTypeID) Equal(other ccfTypeID) bool {
	return id == other
}

// IMPORTANT: Don't use cadence.Type as map key because all Cadence composite/interface
// types are pointers, and different instance of the same type will be treated as
// different map key.
type ccfTypeIDByCadenceType map[string]ccfTypeID

func (types ccfTypeIDByCadenceType) id(t cadence.Type) (ccfTypeID, error) {
	id, ok := types[t.ID()]
	if !ok {
		return 0, fmt.Errorf("CCF type ID not found for type %s", t.ID())
	}
	return id, nil
}

type cadenceTypeByCCFTypeID map[ccfTypeID]cadence.Type

func (ids cadenceTypeByCCFTypeID) add(id ccfTypeID, typ cadence.Type) bool {
	_, ok := ids[id]
	if ok {
		return false
	}
	ids[id] = typ
	return true
}

func (ids cadenceTypeByCCFTypeID) typ(id ccfTypeID) (cadence.Type, error) {
	t, ok := ids[id]
	if !ok {
		return nil, fmt.Errorf("type not found for CCF type ID %d", id)
	}
	return t, nil
}
