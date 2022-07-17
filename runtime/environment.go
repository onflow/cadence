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

package runtime

import (
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type Environment struct {
	baseActivation      *interpreter.VariableActivation
	baseValueActivation *sema.VariableActivation
	Interface           Interface
}

var _ stdlib.Logger = &Environment{}
var _ stdlib.BlockAtHeightProvider = &Environment{}
var _ stdlib.CurrentBlockProvider = &Environment{}

func newEnvironment() *Environment {
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseActivation := interpreter.NewVariableActivation(nil, interpreter.BaseActivation)
	return &Environment{
		baseActivation:      baseActivation,
		baseValueActivation: baseValueActivation,
	}
}

func (e *Environment) Declare(valueDeclaration stdlib.StandardLibraryValue) {
	e.baseValueActivation.DeclareValue(valueDeclaration)
	e.baseActivation.Declare(valueDeclaration)
}

func NewScriptEnvironment(declarations ...stdlib.StandardLibraryValue) *Environment {
	environment := NewBaseEnvironment()
	// TODO: add getAuthAccount
	return environment
}

func NewBaseEnvironment(declarations ...stdlib.StandardLibraryValue) *Environment {
	environment := newEnvironment()
	for _, valueDeclaration := range stdlib.BuiltinValues {
		environment.Declare(valueDeclaration)
	}
	environment.Declare(stdlib.NewLogFunction(environment))
	environment.Declare(stdlib.NewGetBlockFunction(environment))
	environment.Declare(stdlib.NewGetCurrentBlockFunction(environment))
	return environment
}

func (e *Environment) ProgramLog(message string) error {
	return e.Interface.ProgramLog(message)
}

func (e *Environment) GetBlockAtHeight(height uint64) (block stdlib.Block, exists bool, err error) {
	return e.Interface.GetBlockAtHeight(height)
}

func (e *Environment) GetCurrentBlockHeight() (uint64, error) {
	return e.Interface.GetCurrentBlockHeight()
}
