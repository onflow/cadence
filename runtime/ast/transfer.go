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

package ast

import (
	"encoding/json"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
)

// Transfer represents the operation in variable declarations
// and assignments
type Transfer struct {
	Operation TransferOperation
	Pos       Position `json:"-"`
}

func NewTransfer(memoryGauge common.MemoryGauge, operation TransferOperation, position Position) *Transfer {
	common.UseMemory(memoryGauge, common.TransferMemoryUsage)
	return &Transfer{
		Operation: operation,
		Pos:       position,
	}
}

func (f Transfer) StartPosition() Position {
	return f.Pos
}

func (f Transfer) EndPosition(memoryGauge common.MemoryGauge) Position {
	length := len(f.Operation.Operator())
	return f.Pos.Shifted(memoryGauge, length-1)
}

func (f Transfer) MarshalJSON() ([]byte, error) {
	type Alias Transfer
	return json.Marshal(&struct {
		*Alias
		Type string
		Range
	}{
		Type:  "Transfer",
		Range: NewUnmeteredRangeFromPositioned(f),
		Alias: (*Alias)(&f),
	})
}

var copyTransferDoc prettier.Doc = prettier.Text("=")
var moveTransferDoc prettier.Doc = prettier.Text("<-")
var forceMoveTransferDoc prettier.Doc = prettier.Text("<-!")

func (f Transfer) Doc() prettier.Doc {
	switch f.Operation {
	case TransferOperationMove:
		return moveTransferDoc
	case TransferOperationMoveForced:
		return forceMoveTransferDoc
	default:
		return copyTransferDoc
	}
}
