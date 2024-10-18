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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/tests/utils"
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
			inter.Globals.Get("name").GetValue(inter),
		)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("Red"),
			inter.Globals.Get("color").GetValue(inter),
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

	fruitType.Members = &sema.StringMemberOrderedMap{}

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

	fruitStaticType := interpreter.NewCompositeStaticTypeComputeTypeID(
		nil,
		TestLocation,
		fruitType.Identifier,
	)

	fruitValue := interpreter.NewSimpleCompositeValue(
		nil,
		fruitType.ID(),
		fruitStaticType,
		[]string{"name", "color"},
		map[string]interpreter.Value{
			"name": interpreter.NewUnmeteredStringValue("Apple"),
		},
		func(name string, _ *interpreter.Interpreter, _ interpreter.LocationRange) interpreter.Value {
			if name == "color" {
				return interpreter.NewUnmeteredStringValue("Red")
			}

			return nil
		},
		nil,
		nil,
	)

	valueDeclaration := stdlib.StandardLibraryValue{
		Name:  "fruit",
		Type:  fruitType,
		Value: fruitValue,
		Kind:  common.DeclarationKindConstant,
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(valueDeclaration)

	baseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
	baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
		Name: fruitType.Identifier,
		Type: fruitType,
		Kind: common.DeclarationKindStructure,
	})

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, valueDeclaration)

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
				BaseTypeActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseTypeActivation
				},
				CheckHandler: func(checker *sema.Checker, check func()) {
					if checker.Location == TestLocation {
						checker.Elaboration.SetCompositeType(
							fruitType.ID(),
							fruitType,
						)
					}
					check()
				},
			},
			Config: &interpreter.Config{
				Storage: storage,
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
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

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		code := fmt.Sprintf(
			`
              contract C {}

              fun test() {
                  authAccount.storage.save(%s, to: /storage/c)
              }
		    `,
			value,
		)
		inter, _ := testAccountWithErrorHandler(
			t,
			address,
			true,
			nil,
			code,
			sema.Config{},
			func(err error) {
				var invalidMoveError *sema.InvalidMoveError
				require.ErrorAs(t, err, &invalidMoveError)
			})

		_, err := inter.Invoke("test")
		RequireError(t, err)

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
