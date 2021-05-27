/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretTransferCheck(t *testing.T) {

	t.Parallel()

	ty := &sema.CompositeType{
		Location:   utils.TestLocation,
		Identifier: "Fruit",
		Kind:       common.CompositeKindStructure,
	}

	valueDeclarations := stdlib.StandardLibraryValues{
		{
			Name: "fruit",
			Type: ty,
			// NOTE: not an instance of the type
			Value: interpreter.NewStringValue("fruit"),
			Kind:  common.DeclarationKindConstant,
		},
	}

	typeDeclarations := stdlib.StandardLibraryTypes{
		{
			Name: ty.Identifier,
			Type: ty,
			Kind: common.DeclarationKindStructure,
		},
	}

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          fun test() {
            let alsoFruit: Fruit = fruit
          }
        `,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations.ToSemaValueDeclarations()),
				sema.WithPredeclaredTypes(typeDeclarations.ToTypeDeclarations()),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(valueDeclarations.ToInterpreterValueDeclarations()),
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.Error(t, err)

	require.ErrorAs(t, err, &interpreter.ValueTransferTypeError{})
}
