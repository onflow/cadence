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
	"strings"
)

type Argument struct {
	Label                string    `json:",omitempty"`
	LabelStartPos        *Position `json:",omitempty"`
	LabelEndPos          *Position `json:",omitempty"`
	TrailingSeparatorPos Position
	Expression           Expression
}

func (a *Argument) StartPosition() Position {
	if a.LabelStartPos != nil {
		return *a.LabelStartPos
	}
	return a.Expression.StartPosition()
}

func (a *Argument) EndPosition() Position {
	return a.Expression.EndPosition()
}

func (a *Argument) String() string {
	var builder strings.Builder
	if a.Label != "" {
		builder.WriteString(a.Label)
		builder.WriteString(": ")
	}
	builder.WriteString(a.Expression.String())
	return builder.String()
}

func (a *Argument) MarshalJSON() ([]byte, error) {
	type Alias Argument
	return json.Marshal(&struct {
		Range
		*Alias
	}{
		Range: NewRangeFromPositioned(a),
		Alias: (*Alias)(a),
	})
}
