/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
