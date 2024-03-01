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
	"github.com/onflow/atree"
	"github.com/onflow/flow-go/cmd/util/ledger/util"
	"github.com/onflow/flow-go/model/flow"
)

type PayloadSnapshotLedger struct {
	*util.PayloadSnapshot
}

var _ atree.Ledger = PayloadSnapshotLedger{}

func (p PayloadSnapshotLedger) GetValue(owner, key []byte) (value []byte, err error) {
	registerID := flow.NewRegisterID(flow.Address(owner), string(key))
	return p.PayloadSnapshot.Get(registerID)
}

func (p PayloadSnapshotLedger) ValueExists(owner, key []byte) (exists bool, err error) {
	registerID := flow.NewRegisterID(flow.Address(owner), string(key))
	_, exists = p.Payloads[registerID]
	return
}

func (PayloadSnapshotLedger) SetValue(_, _, _ []byte) (err error) {
	panic(atree.NewUnreachableError())
}

func (PayloadSnapshotLedger) AllocateStorageIndex(_ []byte) (atree.StorageIndex, error) {
	panic(atree.NewUnreachableError())
}
