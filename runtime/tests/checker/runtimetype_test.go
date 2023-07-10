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

func TestCheckOptionalTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "String",
			code: `
              let result = OptionalType(Type<String>())
            `,
			expectedError: nil,
		},
		{
			name: "Int",
			code: `
				let result = OptionalType(Type<Int>())
            `,
			expectedError: nil,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = OptionalType(Type<@R>())
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch",
			code: `
              let result = OptionalType(3)
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = OptionalType(Type<Int>(), Type<Int>())
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "too few args",
			code: `
              let result = OptionalType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckVariableSizedArrayTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "String",
			code: `
              let result = VariableSizedArrayType(Type<String>())
            `,
			expectedError: nil,
		},
		{
			name: "Int",
			code: `
				let result = VariableSizedArrayType(Type<Int>())
            `,
			expectedError: nil,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = VariableSizedArrayType(Type<@R>())
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch",
			code: `
              let result = VariableSizedArrayType(3)
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = VariableSizedArrayType(Type<Int>(), Type<Int>())
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "too few args",
			code: `
              let result = VariableSizedArrayType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckConstantSizedArrayTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "String",
			code: `
              let result = ConstantSizedArrayType(type: Type<String>(), size: 3)
            `,
			expectedError: nil,
		},
		{
			name: "Int",
			code: `
				let result = ConstantSizedArrayType(type: Type<Int>(), size: 2)
            `,
			expectedError: nil,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = ConstantSizedArrayType(type: Type<@R>(), size: 4)
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = ConstantSizedArrayType(type: 3, size: 4)
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = ConstantSizedArrayType(type: Type<Int>(), size: "")
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = ConstantSizedArrayType(type:Type<Int>(), size: 3, 4)
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "one arg",
			code: `
              let result = ConstantSizedArrayType(type: Type<Int>())
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "no args",
			code: `
              let result = ConstantSizedArrayType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "second label missing",
			code: `
              let result = ConstantSizedArrayType(type: Type<String>(), 3)
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
		{
			name: "first label missing",
			code: `
              let result = ConstantSizedArrayType(Type<String>(), size: 3)
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckDictionaryTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "String/Int",
			code: `
              let result = DictionaryType(key: Type<String>(), value: Type<Int>())
            `,
			expectedError: nil,
		},
		{
			name: "Int/String",
			code: `
				let result = DictionaryType(key: Type<Int>(), value: Type<String>())
            `,
			expectedError: nil,
		},
		{
			name: "resource/struct",
			code: `
              resource R {}
			  struct S {}
              let result = DictionaryType(key: Type<@R>(), value: Type<S>())
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = DictionaryType(key: 3, value: Type<String>())
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "type mismatch second arg",
			code: `
			let result = DictionaryType(key: Type<String>(), value: "")
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = DictionaryType(key: Type<Int>(), value: Type<Int>(), 4)
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "one arg",
			code: `
              let result = DictionaryType(key: Type<Int>())
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "no args",
			code: `
              let result = DictionaryType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "first label missing",
			code: `
              let result = DictionaryType(Type<String>(), value: Type<Int>())
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
		{
			name: "second label missing",
			code: `
              let result = DictionaryType(key: Type<String>(), Type<Int>())
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckCompositeTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "R",
			code: `
              let result = CompositeType("R")
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch",
			code: `
              let result = CompositeType(3)
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = CompositeType("", 3)
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "no args",
			code: `
              let result = CompositeType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckInterfaceTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "R",
			code: `
              let result = InterfaceType("R")
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch",
			code: `
              let result = InterfaceType(3)
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = InterfaceType("", 3)
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "no args",
			code: `
              let result = InterfaceType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckFunctionTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "(String): Int",
			code: `
              let result = FunctionType(parameters: [Type<String>()], return: Type<Int>())
            `,
			expectedError: nil,
		},
		{
			name: "(String, Int): Bool",
			code: `
				let result = FunctionType(parameters: [Type<String>(), Type<Int>()], return: Type<Bool>())
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = FunctionType(parameters: Type<String>(), return: Type<String>())
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "type mismatch nested first arg",
			code: `
              let result = FunctionType(parameters: [Type<String>(), 3], return: Type<String>())
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = FunctionType(parameters: [Type<String>(), Type<Int>()], return: "")
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = FunctionType(parameters: [Type<String>(), Type<Int>()], return: Type<String>(), 4)
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "one arg",
			code: `
              let result = FunctionType(parameters: [Type<String>(), Type<Int>()])
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "no args",
			code: `
              let result = FunctionType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "first label missing",
			code: `
              let result = FunctionType([Type<String>()], return: Type<Int>())
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
		{
			name: "second label missing",
			code: `
              let result = FunctionType(parameters: [Type<String>()], Type<Int>())
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckReferenceTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "auth(X, Y) &R",
			code: `
			  resource R {}
              let result = ReferenceType(entitlements: ["S.test.X", "S.test.Y"], type: Type<@R>())
            `,
			expectedError: nil,
		},
		{
			name: "&String",
			code: `
              let result = ReferenceType(entitlements: [], type: Type<String>())
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = ReferenceType(entitlements: "", type: Type<Int>())
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = ReferenceType(entitlements: [], type: "")
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = ReferenceType(entitlements: [], type: Type<String>(), Type<Int>())
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "one arg",
			code: `
              let result = ReferenceType(entitlements: [])
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "no args",
			code: `
              let result = ReferenceType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "first label missing",
			code: `
			  resource R {}
              let result = ReferenceType([], type: Type<@R>())
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
		{
			name: "second label missing",
			code: `
			  resource R {}
              let result = ReferenceType(entitlements: [], Type<@R>())
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					sema.NewOptionalType(nil, sema.MetaType),
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckIntersectionTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "S{I1, I2}",
			code: `
              let result = IntersectionType(identifier: "S", types: ["I1", "I2"])
            `,
			expectedError: nil,
		},
		{
			name: "S{}",
			code: `
              struct S {}
              let result = IntersectionType(identifier: "S", types: [])
            `,
			expectedError: nil,
		},
		{
			name: "{S}",
			code: `
              struct S {}
              let result = IntersectionType(identifier: nil, types: ["S"])
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = IntersectionType(identifier: 3, types: ["I"])
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = IntersectionType(identifier: "A", types: [3])
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = IntersectionType(identifier: "A", types: ["I1"], ["I2"])
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "one arg",
			code: `
              let result = IntersectionType(identifier: "A")
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "no args",
			code: `
              let result = IntersectionType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "missing first label",
			code: `
              let result = IntersectionType("S", types: ["I1", "I2"])
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
		{
			name: "missing second label",
			code: `
              let result = IntersectionType(identifier: "S", ["I1", "I2"])
            `,
			expectedError: &sema.MissingArgumentLabelError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}

func TestCheckCapabilityTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name          string
		code          string
		expectedError error
	}{
		{
			name: "&String",
			code: `
              let result = CapabilityType(Type<&String>())
            `,
			expectedError: nil,
		},
		{
			name: "&Int",
			code: `
              let result = CapabilityType(Type<&Int>())
            `,
			expectedError: nil,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = CapabilityType(Type<@R>())
            `,
			expectedError: nil,
		},
		{
			name: "type mismatch",
			code: `
              let result = CapabilityType(3)
            `,
			expectedError: &sema.TypeMismatchError{},
		},
		{
			name: "too many args",
			code: `
              let result = CapabilityType(Type<Int>(), Type<Int>())
            `,
			expectedError: &sema.ArgumentCountError{},
		},
		{
			name: "too few args",
			code: `
              let result = CapabilityType()
            `,
			expectedError: &sema.ArgumentCountError{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, testCase.expectedError, errs[0])
			}
		})
	}
}
