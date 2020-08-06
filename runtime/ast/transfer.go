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

package ast

import (
	"encoding/json"
)

// Transfer represents the operation in variable declarations
// and assignments
//
type Transfer struct {
	Operation TransferOperation
	Pos       Position `json:"-"`
}

func (f Transfer) StartPosition() Position {
	return f.Pos
}

func (f Transfer) EndPosition() Position {
	length := len(f.Operation.Operator())
	return f.Pos.Shifted(length - 1)
}

func (f Transfer) MarshalJSON() ([]byte, error) {
	type Alias Transfer
	return json.Marshal(&struct {
		Type string
		Range
		*Alias
	}{
		Type:  "Transfer",
		Range: NewRangeFromPositioned(f),
		Alias: (*Alias)(&f),
	})
}
