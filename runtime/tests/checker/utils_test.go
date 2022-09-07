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

package checker

import (
	"testing"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func ParseAndCheckWithPanic(t *testing.T, code string) (*sema.Checker, error) {

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.PanicFunction)

	return ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivation: baseValueActivation,
			},
		},
	)
}

func ParseAndCheckWithAny(t *testing.T, code string) (*sema.Checker, error) {

	baseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
	baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
		Name: "Any",
		Type: sema.AnyType,
		Kind: common.DeclarationKindType,
	})

	return ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseTypeActivation: baseTypeActivation,
			},
		},
	)
}
