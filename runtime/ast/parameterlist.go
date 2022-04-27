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

import "sync"

type ParameterList struct {
	once                    sync.Once
	Parameters              []*Parameter
	_parametersByIdentifier map[string]*Parameter
	Range
}

// EffectiveArgumentLabels returns the effective argument labels that
// the arguments of a call must use:
// If no argument label is declared for parameter,
// the parameter name is used as the argument label
//
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
