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

		require.Equal(t,
			interpreter.NewStringValue("Apple"),
			inter.Globals.Get("name").GetValue(),
		)

		require.Equal(t,
			interpreter.NewStringValue("Red"),
			inter.Globals.Get("color").GetValue(),
		)
	})
}

// Utility methods
func testCompositeValue(t *testing.T, code string) *interpreter.Interpreter {

	var valueDeclarations stdlib.StandardLibraryValues

	// 'fruit' composite type
	fruitType := &sema.CompositeType{
		Location:   utils.TestLocation,
		Identifier: "fruit",
		Kind:       common.CompositeKindStructure,
	}

	fruitType.Members = sema.NewStringMemberOrderedMap()

	fruitType.Members.Set("name", sema.NewPublicConstantFieldMember(
		fruitType,
		"name",
		sema.StringType,
		"This is the name",
	))

	fruitType.Members.Set("color", sema.NewPublicConstantFieldMember(
		fruitType,
		"color",
		sema.StringType,
		"This is the color",
	))

	fields := interpreter.NewStringValueOrderedMap()
	fields.Set("name", interpreter.NewStringValue("Apple"))

	value := interpreter.NewCompositeValue(
		utils.TestLocation,
		fruitType.Identifier,
		common.CompositeKindStructure,
		fields,
		nil,
	)

	value.ComputedFields = interpreter.NewStringComputedFieldOrderedMap()
	value.ComputedFields.Set("color", func(*interpreter.Interpreter) interpreter.Value {
		return interpreter.NewStringValue("Red")
	})

	customStructValue := stdlib.StandardLibraryValue{
		Name:  value.QualifiedIdentifier(),
		Type:  fruitType,
		Value: value,
		Kind:  common.DeclarationKindConstant,
	}

	valueDeclarations = append(valueDeclarations, customStructValue)

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations.ToSemaValueDeclarations()),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(valueDeclarations.ToInterpreterValueDeclarations()),
			},
		},
	)
	require.NoError(t, err)

	return inter
}
