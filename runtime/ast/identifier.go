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

	"github.com/onflow/cadence/runtime/common"
)

// Identifier

type Identifier struct {
	Identifier string
	Pos        Position
}

func NewIdentifier(memoryGauge common.MemoryGauge, identifier string, pos Position) Identifier {
	common.UseMemory(memoryGauge, common.IdentifierMemoryUsage)
	return Identifier{
		Identifier: identifier,
		Pos:        pos,
	}
}

func NewEmptyIdentifier(memoryGauge common.MemoryGauge, pos Position) Identifier {
	common.UseMemory(memoryGauge, common.IdentifierMemoryUsage)
	return Identifier{
		Pos: pos,
	}
}

func (i Identifier) String() string {
	return i.Identifier
}

func (i Identifier) StartPosition() Position {
	return i.Pos
}

func (i Identifier) EndPosition(memoryGauge common.MemoryGauge) Position {
	length := len(i.Identifier)
	return i.Pos.Shifted(memoryGauge, length-1)
}

func (i Identifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Identifier string
		Range
	}{
		Identifier: i.Identifier,
		Range:      NewUnmeteredRangeFromPositioned(i),
	})
}
