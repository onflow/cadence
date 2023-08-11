/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckMetaType(t *testing.T) {

	t.Parallel()

	t.Run("constructor", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let type: Type = Type<[Int]>()
        `)

		require.NoError(t, err)

		assert.Equal(t,
			sema.MetaType,
			RequireGlobalValue(t, checker.Elaboration, "type"),
		)
	})

	t.Run("identifier", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let type = Type<[Int]>()
          let identifier = type.identifier
        `)

		require.NoError(t, err)

		assert.Equal(t,
			sema.MetaType,
			RequireGlobalValue(t, checker.Elaboration, "type"),
		)
	})
}

func TestCheckIsInstance(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name              string
		code              string
		expectedErrorType error
	}{
		{
			name: "string is an instance of string",
			code: `
              let stringType = Type<String>()
              let result = "abc".isInstance(stringType)
            `,
			expectedErrorType: nil,
		},
		{
			name: "int is an instance of int",
			code: `
              let intType = Type<Int>()
              let result = (1).isInstance(intType)
            `,
			expectedErrorType: nil,
		},
		{
			name: "resource is an instance of resource",
			code: `
              resource R {}

              let r <- create R()
              let rType = Type<@R>()
              let result = r.isInstance(rType)
            `,
			expectedErrorType: nil,
		},
		{
			name: "1 is an instance of Int?",
			code: `
              let result = (1).isInstance(Type<Int?>())
            `,
			expectedErrorType: nil,
		},
		{
			name: "isInstance must take a type",
			code: `
              let result = (1).isInstance(3)
            `,
			expectedErrorType: &sema.TypeMismatchError{},
		},
		{
			name: "nil is not a type",
			code: `
              let result = (1).isInstance(nil)
            `,
			expectedErrorType: &sema.TypeMismatchError{},
		},
		{
			name: "argument label",
			code: `
              let result = (1).isInstance(type: Type<Int>())
            `,
			expectedErrorType: &sema.IncorrectArgumentLabelError{},
		},
		{
			name: "too many arguments",
			code: `
              let result = (1).isInstance(Type<Int>(), Type<Int>())
            `,
			expectedErrorType: &sema.ExcessiveArgumentsError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)
			if testCase.expectedErrorType == nil {
				require.NoError(t, err)
				assert.Equal(t,
					sema.BoolType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, testCase.expectedErrorType, errs[0])
			}
		})
	}
}

func TestCheckIsSubtype(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name              string
		code              string
		expectedErrorType error
	}{
		{
			name: "string is a subtype of string",
			code: `
              let stringType = Type<String>()
              let result = stringType.isSubtype(of: stringType)
            `,
			expectedErrorType: nil,
		},
		{
			name: "int is a subtype of int",
			code: `
              let intType = Type<Int>()
              let result = intType.isSubtype(of: intType)
            `,
			expectedErrorType: nil,
		},
		{
			name: "resource is a subtype of resource",
			code: `
              resource R {}
              let rType = Type<@R>()
              let result = rType.isSubtype(of: rType)
            `,
			expectedErrorType: nil,
		},
		{
			name: "Int is an instance of Int?",
			code: `
              let result = Type<Int>().isSubtype(of: Type<Int?>())
            `,
			expectedErrorType: nil,
		},
		{
			name: "isSubtype must take a type",
			code: `
              let result = Type<Int>().isSubtype(of: 3)
            `,
			expectedErrorType: &sema.TypeMismatchError{},
		},
		{
			name: "isSubtype must take an argument",
			code: `
              let result = Type<Int>().isSubtype()
            `,
			expectedErrorType: &sema.InsufficientArgumentsError{},
		},
		{
			name: "isSubtype argument must be named",
			code: `
              let result = Type<Int>().isSubtype(Type<Int?>())
            `,
			expectedErrorType: &sema.MissingArgumentLabelError{},
		},
		{
			name: "isSubtype must take fewer than two arguments",
			code: `
              let result = Type<Int>().isSubtype(of: Type<Int?>(), Type<Int?>())
            `,
			expectedErrorType: &sema.ExcessiveArgumentsError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)
			if testCase.expectedErrorType == nil {
				require.NoError(t, err)
				assert.Equal(t,
					sema.BoolType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, testCase.expectedErrorType, errs[0])
			}
		})
	}
}

func TestCheckIsInstance_Redeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct R {
          fun isInstance() {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckGetType(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name string
		code string
	}{
		{
			name: "String",
			code: `
              let result = "abc".getType()
            `,
		},
		{
			name: "Int",
			code: `
              let result = (1).getType()
            `,
		},
		{
			name: "resource",
			code: `
              resource R {}

              let r <- create R()
              let result = r.getType()
            `,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			require.NoError(t, err)
			assert.Equal(t,
				sema.MetaType,
				RequireGlobalValue(t, checker.Elaboration, "result"),
			)
		})
	}
}
