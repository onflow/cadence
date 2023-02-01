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

import "github.com/onflow/cadence/runtime/common"

type TypeParameter struct {
	Identifier Identifier
	TypeBound  *TypeAnnotation
}

var _ HasPosition = &TypeParameter{}

func NewTypeParameter(
	gauge common.MemoryGauge,
	identifier Identifier,
	typeBound *TypeAnnotation,
) *TypeParameter {
	common.UseMemory(gauge, common.TypeParameterMemoryUsage)
	return &TypeParameter{
		Identifier: identifier,
		TypeBound:  typeBound,
	}
}

func (t *TypeParameter) StartPosition() Position {
	return t.Identifier.StartPosition()
}

func (t *TypeParameter) EndPosition(memoryGauge common.MemoryGauge) Position {
	if t.TypeBound != nil {
		return t.TypeBound.EndPosition(memoryGauge)
	}
	return t.Identifier.EndPosition(memoryGauge)
}
