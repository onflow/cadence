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
	"sync"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
)

type TypeParameterList struct {
	once                        sync.Once
	TypeParameters              []*TypeParameter
	_typeParametersByIdentifier map[string]*TypeParameter
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
	l.once.Do(l.initialize)
	return l._typeParametersByIdentifier
}

func (l *TypeParameterList) initialize() {
	typeParametersByIdentifier := make(map[string]*TypeParameter, len(l.TypeParameters))
	for _, typeParameter := range l.TypeParameters {
		typeParametersByIdentifier[typeParameter.Identifier.Identifier] = typeParameter
	}
	l._typeParametersByIdentifier = typeParametersByIdentifier
}

func (l *TypeParameterList) IsEmpty() bool {
	return l == nil || len(l.TypeParameters) == 0
}

var typeParameterSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (l *TypeParameterList) Doc() prettier.Doc {

	if len(l.TypeParameters) == 0 {
		return nil
	}

	typeParameterDocs := make([]prettier.Doc, 0, len(l.TypeParameters))

	for _, typeParameter := range l.TypeParameters {
		var parameterDoc prettier.Concat

		parameterDoc = append(
			parameterDoc,
			prettier.Text(typeParameter.Identifier.Identifier),
		)

		if typeParameter.TypeBound != nil {
			parameterDoc = append(
				parameterDoc,
				typeSeparatorSpaceDoc,
				typeParameter.TypeBound.Doc(),
			)
		}

		typeParameterDocs = append(typeParameterDocs, parameterDoc)
	}

	return prettier.WrapParentheses(
		prettier.Join(
			parameterSeparatorDoc,
			typeParameterDocs...,
		),
		prettier.SoftLine{},
	)
}

func (l *TypeParameterList) String() string {
	return Prettier(l)
}
