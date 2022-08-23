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
		func(name string, _ *interpreter.Interpreter, _ func() interpreter.LocationRange) interpreter.Value {
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

	baseActivation := interpreter.NewVariableActivation(nil, interpreter.BaseActivation)
	baseActivation.Declare(valueDeclaration)

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivation: baseValueActivation,
				BaseTypeActivation:  baseTypeActivation,
				CheckHandler: func(checker *sema.Checker, check func()) {
					if checker.Location == TestLocation {
						checker.Elaboration.CompositeTypes[fruitType.ID()] = fruitType
					}
					check()
				},
			},
			Config: &interpreter.Config{
				Storage:        storage,
				BaseActivation: baseActivation,
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
