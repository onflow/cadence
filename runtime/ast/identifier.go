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

package ast

import (
	"encoding/json"
)

// Identifier

type Identifier struct {
	Identifier string
	Pos        Position
}

func (i Identifier) String() string {
	return i.Identifier
}

func (i Identifier) StartPosition() Position {
	return i.Pos
}

func (i Identifier) EndPosition() Position {
	length := len(i.Identifier)
	return i.Pos.Shifted(length - 1)
}

func (i Identifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Identifier string
		Range
	}{
		Identifier: i.Identifier,
		Range:      NewRangeFromPositioned(i),
	})
}
