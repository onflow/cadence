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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretCompositeValue(t *testing.T) {

	t.Parallel()

	t.Run("computed fields", func(t *testing.T) {

		t.Parallel()

		inter := testCompositeValue(
			t,
			`
              // Get a static field using member access
              let name: String = fruit.name

              // Get a computed field using member access
              let color: String = fruit.color
            `,
		)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Apple"),
			inter.Globals["name"].GetValue(),
		)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Red"),
			inter.Globals["color"].GetValue(),
		)
	})
}

// Utility methods
func testCompositeValue(t *testing.T, code string) *interpreter.Interpreter {

	storage := newUnmeteredInMemoryStorage()

	// 'fruit' composite type
	fruitType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "Fruit",
		Kind:       common.CompositeKindStructure,
	}

	fruitType.Members = sema.NewStringMemberOrderedMap()

	fruitType.Members.Set("name", sema.NewUnmeteredPublicConstantFieldMember(
		fruitType,
		"name",
		sema.StringType,
		"This is the name",
	))

	fruitType.Members.Set("color", sema.NewUnmeteredPublicConstantFieldMember(
		fruitType,
		"color",
		sema.StringType,
		"This is the color",
	))

	valueDeclarations := stdlib.StandardLibraryValues{
		{
			Name: "fruit",
			Type: fruitType,
			ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				fields := []interpreter.CompositeField{
					{
						Name:  "name",
						Value: interpreter.NewUnmeteredStringValue("Apple"),
					},
				}

				value := interpreter.NewCompositeValue(
					inter,
					TestLocation,
					fruitType.Identifier,
					common.CompositeKindStructure,
					fields,
					common.Address{},
				)

				value.ComputedFields = map[string]interpreter.ComputedField{
					"color": func(_ *interpreter.Interpreter, _ func() interpreter.LocationRange) interpreter.Value {
						return interpreter.NewUnmeteredStringValue("Red")
					},
				}

				return value
			},
			Kind: common.DeclarationKindConstant,
		},
	}

	typeDeclarations := []sema.TypeDeclaration{
		stdlib.StandardLibraryType{
			Name: fruitType.Identifier,
			Type: fruitType,
			Kind: common.DeclarationKindStructure,
		},
	}

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations.ToSemaValueDeclarations()),
				sema.WithPredeclaredTypes(typeDeclarations),
			},
			Options: []interpreter.Option{
				interpreter.WithStorage(storage),
				interpreter.WithPredeclaredValues(valueDeclarations.ToInterpreterValueDeclarations()),
			},
		},
	)
	require.NoError(t, err)

	return inter
}

func TestInterpretContractTransfer(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, value string) {

		t.Parallel()

		address := interpreter.NewAddressValueFromBytes([]byte{42})

		code := fmt.Sprintf(
			`
              contract C {}

              fun test() {
                  authAccount.save(%s, to: /storage/c)
              }
		    `,
			value,
		)
		inter, _ := testAccount(t, address, true, code)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		var nonTransferableValueError interpreter.NonTransferableValueError
		require.ErrorAs(t, err, &nonTransferableValueError)
	}

	t.Run("simple", func(t *testing.T) {
		test(t, "C as AnyStruct")
	})

	t.Run("nested", func(t *testing.T) {
		test(t, "[C as AnyStruct]")
	})
}
