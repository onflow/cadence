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
		name  string
		code  string
		valid bool
	}{
		{
			name: "String",
			code: `
              let result = OptionalType(Type<String>())
            `,
			valid: true,
		},
		{
			name: "Int",
			code: `
				let result = OptionalType(Type<Int>())
            `,
			valid: true,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = OptionalType(Type<@R>())
            `,
			valid: true,
		},
		{
			name: "type mismatch",
			code: `
              let result = OptionalType(3)
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = OptionalType(Type<Int>(), Type<Int>())
            `,
			valid: false,
		},
		{
			name: "too few args",
			code: `
              let result = OptionalType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckVariableSizedArrayTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "String",
			code: `
              let result = VariableSizedArrayType(Type<String>())
            `,
			valid: true,
		},
		{
			name: "Int",
			code: `
				let result = VariableSizedArrayType(Type<Int>())
            `,
			valid: true,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = VariableSizedArrayType(Type<@R>())
            `,
			valid: true,
		},
		{
			name: "type mismatch",
			code: `
              let result = VariableSizedArrayType(3)
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = VariableSizedArrayType(Type<Int>(), Type<Int>())
            `,
			valid: false,
		},
		{
			name: "too few args",
			code: `
              let result = VariableSizedArrayType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckConstantSizedArrayTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "String",
			code: `
              let result = ConstantSizedArrayType(Type<String>(), 3)
            `,
			valid: true,
		},
		{
			name: "Int",
			code: `
				let result = ConstantSizedArrayType(Type<Int>(), 2)
            `,
			valid: true,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = ConstantSizedArrayType(Type<@R>(), 4)
            `,
			valid: true,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = ConstantSizedArrayType(3, 4)
            `,
			valid: false,
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = ConstantSizedArrayType(Type<Int>(), "")
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = ConstantSizedArrayType(Type<Int>(), 3, 4)
            `,
			valid: false,
		},
		{
			name: "one arg",
			code: `
              let result = ConstantSizedArrayType(Type<Int>())
            `,
			valid: false,
		},
		{
			name: "no args",
			code: `
              let result = ConstantSizedArrayType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckDictionaryTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "String/Int",
			code: `
              let result = DictionaryType(Type<String>(), Type<Int>())
            `,
			valid: true,
		},
		{
			name: "Int/String",
			code: `
				let result = DictionaryType(Type<Int>(), Type<String>())
            `,
			valid: true,
		},
		{
			name: "resource/struct",
			code: `
              resource R {}
			  struct S {}
              let result = DictionaryType(Type<@R>(), Type<S>())
            `,
			valid: true,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = DictionaryType(3, Type<String>())
            `,
			valid: false,
		},
		{
			name: "type mismatch second arg",
			code: `
			let result = DictionaryType(Type<String>(), "")
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = DictionaryType(Type<Int>(), Type<Int>(), 4)
            `,
			valid: false,
		},
		{
			name: "one arg",
			code: `
              let result = DictionaryType(Type<Int>())
            `,
			valid: false,
		},
		{
			name: "no args",
			code: `
              let result = DictionaryType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckCompositeTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "R",
			code: `
              let result = CompositeType("R")
            `,
			valid: true,
		},
		{
			name: "type mismatch",
			code: `
              let result = DictionaryType(3)
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = DictionaryType("", 3)
            `,
			valid: false,
		},
		{
			name: "no args",
			code: `
              let result = DictionaryType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckInterfaceTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "R",
			code: `
              let result = InterfaceType("R")
            `,
			valid: true,
		},
		{
			name: "type mismatch",
			code: `
              let result = InterfaceType(3)
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = InterfaceType("", 3)
            `,
			valid: false,
		},
		{
			name: "no args",
			code: `
              let result = InterfaceType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckFunctionTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "(String): Int",
			code: `
              let result = FunctionType([Type<String>()], Type<Int>())
            `,
			valid: true,
		},
		{
			name: "(String, Int): Bool",
			code: `
				let result = FunctionType([Type<String>(), Type<Int>()], Type<Bool>())
            `,
			valid: true,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = FunctionType(Type<String>(), Type<String>())
            `,
			valid: false,
		},
		{
			name: "type mismatch nested first arg",
			code: `
              let result = FunctionType([Type<String>(), 3], Type<String>())
            `,
			valid: false,
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = FunctionType([Type<String>(), Type<Int>()], "")
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = FunctionType([Type<String>(), Type<Int>()], Type<String>(), 4)
            `,
			valid: false,
		},
		{
			name: "one arg",
			code: `
              let result = FunctionType([Type<String>(), Type<Int>()])
            `,
			valid: false,
		},
		{
			name: "no args",
			code: `
              let result = FunctionType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckReferenceTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "auth &R",
			code: `
			  resource R {}
              let result = ReferenceType(true, Type<@R>())
            `,
			valid: true,
		},
		{
			name: "&String",
			code: `
				let result = ReferenceType(false, Type<String>())
            `,
			valid: true,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = ReferenceType("", Type<Int>())
            `,
			valid: false,
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = ReferenceType(true, "")
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = ReferenceType(true, Type<String>(), Type<Int>())
            `,
			valid: false,
		},
		{
			name: "one arg",
			code: `
              let result = ReferenceType(true)
            `,
			valid: false,
		},
		{
			name: "no args",
			code: `
              let result = ReferenceType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					sema.MetaType,
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckRestrictedTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "S{I1, I2}",
			code: `
              let result = RestrictedType("S", ["I1", "I2"])
            `,
			valid: true,
		},
		{
			name: "S{}",
			code: `
				struct S {}
				let result = RestrictedType("S", [])
            `,
			valid: true,
		},
		{
			name: "{S}",
			code: `
				struct S {}
				let result = RestrictedType(nil, ["S"])
            `,
			valid: true,
		},
		{
			name: "type mismatch first arg",
			code: `
              let result = RestrictedType(3, ["I"])
            `,
			valid: false,
		},
		{
			name: "type mismatch second arg",
			code: `
              let result = RestrictedType("A", [3])
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = RestrictedType("A", ["I1"], [])
            `,
			valid: false,
		},
		{
			name: "one arg",
			code: `
              let result = RestrictedType("A")
            `,
			valid: false,
		},
		{
			name: "no args",
			code: `
              let result = RestrictedType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCheckCapabilityTypeConstructor(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name: "&String",
			code: `
              let result = CapabilityType(Type<&String>())
            `,
			valid: true,
		},
		{
			name: "&Int",
			code: `
				let result = CapabilityType(Type<&Int>())
            `,
			valid: true,
		},
		{
			name: "resource",
			code: `
              resource R {}
              let result = CapabilityType(Type<@R>())
            `,
			valid: true,
		},
		{
			name: "type mismatch",
			code: `
              let result = CapabilityType(3)
            `,
			valid: false,
		},
		{
			name: "too many args",
			code: `
              let result = CapabilityType(Type<Int>(), Type<Int>())
            `,
			valid: false,
		},
		{
			name: "too few args",
			code: `
              let result = CapabilityType()
            `,
			valid: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, testCase.code)

			if testCase.valid {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.OptionalType{Type: sema.MetaType},
					RequireGlobalValue(t, checker.Elaboration, "result"),
				)
			} else {
				require.Error(t, err)
			}
		})
	}
}
