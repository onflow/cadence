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

package ast

import (
	"sync/atomic"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/common"
)

type TypeParameterList struct {
	TypeParameters             []*TypeParameter
	typeParametersByIdentifier atomic.Pointer[map[string]*TypeParameter]
	Range
}

func NewTypeParameterList(
	gauge common.MemoryGauge,
	typeParameters []*TypeParameter,
	astRange Range,
) *TypeParameterList {
	common.UseMemory(gauge, common.ParameterListMemoryUsage)
	return &TypeParameterList{
		TypeParameters: typeParameters,
		Range:          astRange,
	}
}

func (l *TypeParameterList) TypeParametersByIdentifier() map[string]*TypeParameter {
	// Return cached map if already computed
	if cachedMap := l.typeParametersByIdentifier.Load(); cachedMap != nil {
		return *cachedMap
	}

	// Compute map and cache it
	computedMap := make(map[string]*TypeParameter, len(l.TypeParameters))
	for _, typeParameter := range l.TypeParameters {
		computedMap[typeParameter.Identifier.Identifier] = typeParameter
	}
	l.typeParametersByIdentifier.Store(&computedMap)
	return computedMap
}

func (l *TypeParameterList) IsEmpty() bool {
	return l == nil || len(l.TypeParameters) == 0
}

func (l *TypeParameterList) Doc() prettier.Doc {

	if len(l.TypeParameters) == 0 {
		return prettier.Text("")
	}

	typeParameterDocs := make([]prettier.Doc, 0, len(l.TypeParameters))

	for _, typeParameter := range l.TypeParameters {
		typeParameterDocs = append(
			typeParameterDocs,
			docOrEmpty(typeParameter),
		)
	}

	return prettier.Wrap(
		prettier.Text("<"),
		prettier.Join(
			parameterSeparatorDoc,
			typeParameterDocs...,
		),
		prettier.Text(">"),
		prettier.SoftLine{},
	)
}

func (l *TypeParameterList) String() string {
	return Prettier(l)
}
