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
}

func newEnvironment(declarations ...stdlib.StandardLibraryValue) Environment {
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseActivation := interpreter.NewVariableActivation(nil, interpreter.BaseActivation)

	for _, valueDeclaration := range declarations {
		baseValueActivation.DeclareValue(valueDeclaration)
		baseActivation.Declare(valueDeclaration)
	}

	return Environment{
		baseActivation:      baseActivation,
		baseValueActivation: baseValueActivation,
	}
}

func NewScriptEnvironment(declarations ...stdlib.StandardLibraryValue) Environment {
	return newEnvironment(append(declarations, stdlib.BuiltinValues...)...)
}

func NewTransactionEnvironment(declarations ...stdlib.StandardLibraryValue) Environment {
	return newEnvironment(append(declarations, stdlib.BuiltinValues...)...)
}
