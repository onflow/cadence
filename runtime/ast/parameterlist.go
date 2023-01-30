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

type ParameterList struct {
	_parametersByIdentifier map[string]*Parameter
	Parameters              []*Parameter
	Range
	once sync.Once
}

func NewParameterList(
	gauge common.MemoryGauge,
	parameters []*Parameter,
	astRange Range,
) *ParameterList {
	common.UseMemory(gauge, common.ParameterListMemoryUsage)
	return &ParameterList{
		Parameters: parameters,
		Range:      astRange,
	}
}

// EffectiveArgumentLabels returns the effective argument labels that
// the arguments of a call must use:
// If no argument label is declared for parameter,
// the parameter name is used as the argument label
func (l *ParameterList) EffectiveArgumentLabels() []string {
	argumentLabels := make([]string, len(l.Parameters))

	for i, parameter := range l.Parameters {
		argumentLabels[i] = parameter.EffectiveArgumentLabel()
	}

	return argumentLabels
}

func (l *ParameterList) ParametersByIdentifier() map[string]*Parameter {
	l.once.Do(l.initialize)
	return l._parametersByIdentifier
}

func (l *ParameterList) initialize() {
	parametersByIdentifier := make(map[string]*Parameter, len(l.Parameters))
	for _, parameter := range l.Parameters {
		parametersByIdentifier[parameter.Identifier.Identifier] = parameter
	}
	l._parametersByIdentifier = parametersByIdentifier
}

func (l *ParameterList) IsEmpty() bool {
	return l == nil || len(l.Parameters) == 0
}

const parameterListEmptyDoc = prettier.Text("()")

var parameterSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (l *ParameterList) Doc() prettier.Doc {

	if len(l.Parameters) == 0 {
		return parameterListEmptyDoc
	}

	parameterDocs := make([]prettier.Doc, 0, len(l.Parameters))

	for _, parameter := range l.Parameters {
		var parameterDoc prettier.Concat

		if parameter.Label != "" {
			parameterDoc = append(
				parameterDoc,
				prettier.Text(parameter.Label),
				prettier.Space,
			)
		}

		parameterDoc = append(
			parameterDoc,
			prettier.Text(parameter.Identifier.Identifier),
			typeSeparatorSpaceDoc,
			parameter.TypeAnnotation.Doc(),
		)

		parameterDocs = append(parameterDocs, parameterDoc)
	}

	return prettier.WrapParentheses(
		prettier.Join(
			parameterSeparatorDoc,
			parameterDocs...,
		),
		prettier.SoftLine{},
	)
}

func (l *ParameterList) String() string {
	return Prettier(l)
}
