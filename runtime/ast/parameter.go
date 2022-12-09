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

	"github.com/onflow/cadence/runtime/common"
)

type Parameter struct {
	Label          string
	Identifier     Identifier
	TypeAnnotation *TypeAnnotation
	StartPos       Position `json:"-"`
}

func NewParameter(
	gauge common.MemoryGauge,
	label string,
	identifier Identifier,
	typeAnnotation *TypeAnnotation,
	startPos Position,
) *Parameter {
	common.UseMemory(gauge, common.ParameterMemoryUsage)
	return &Parameter{
		Label:          label,
		Identifier:     identifier,
		TypeAnnotation: typeAnnotation,
		StartPos:       startPos,
	}
}

var _ HasPosition = &Parameter{}

// EffectiveArgumentLabel returns the effective argument label that
// an argument in a call must use:
// If no argument label is declared for parameter,
// the parameter name is used as the argument label
func (p *Parameter) EffectiveArgumentLabel() string {
	if p.Label != "" {
		return p.Label
	}
	return p.Identifier.Identifier
}

func (p *Parameter) StartPosition() Position {
	return p.StartPos
}

func (p *Parameter) EndPosition(memoryGauge common.MemoryGauge) Position {
	return p.TypeAnnotation.EndPosition(memoryGauge)
}

func (p *Parameter) MarshalJSON() ([]byte, error) {
	type Alias Parameter
	return json.Marshal(&struct {
		Range
		*Alias
	}{
		Range: NewUnmeteredRangeFromPositioned(p),
		Alias: (*Alias)(p),
	})
}
