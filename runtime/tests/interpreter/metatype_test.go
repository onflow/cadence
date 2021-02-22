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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretMetaTypeEquality(t *testing.T) {

	t.Parallel()

	t.Run("Int == Int", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<Int>() == Type<Int>()
        `)

		assert.Equal(t,
			interpreter.BoolValue(true),
			inter.Globals["result"].Value,
		)
	})

	t.Run("Int != String", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<Int>() == Type<String>()
        `)

		assert.Equal(t,
			interpreter.BoolValue(false),
			inter.Globals["result"].Value,
		)
	})

	t.Run("Int != Int?", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<Int>() == Type<Int?>()
        `)

		assert.Equal(t,
			interpreter.BoolValue(false),
			inter.Globals["result"].Value,
		)
	})

	t.Run("&Int == &Int", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<&Int>() == Type<&Int>()
        `)

		assert.Equal(t,
			interpreter.BoolValue(true),
			inter.Globals["result"].Value,
		)
	})

	t.Run("&Int != &String", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<&Int>() == Type<&String>()
        `)

		assert.Equal(t,
			interpreter.BoolValue(false),
			inter.Globals["result"].Value,
		)
	})

	t.Run("Int != unknownType", func(t *testing.T) {

		t.Parallel()

		valueDeclarations := stdlib.StandardLibraryValues{
			{
				Name: "unknownType",
				Type: sema.MetaType,
				Value: interpreter.TypeValue{
					Type: nil,
				},
				Kind: common.DeclarationKindConstant,
			},
		}

		semaValueDeclarations := valueDeclarations.ToSemaValueDeclarations()
		interpreterValueDeclarations := valueDeclarations.ToInterpreterValueDeclarations()

		inter := parseCheckAndInterpretWithOptions(t,
			`
              let result = Type<Int>() == unknownType
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(semaValueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(interpreterValueDeclarations),
				},
			},
		)

		assert.Equal(t,
			interpreter.BoolValue(false),
			inter.Globals["result"].Value,
		)
	})

	t.Run("unknownType1 != unknownType2", func(t *testing.T) {

		t.Parallel()

		valueDeclarations := stdlib.StandardLibraryValues{
			{
				Name: "unknownType1",
				Type: sema.MetaType,
				Value: interpreter.TypeValue{
					Type: nil,
				},
				Kind: common.DeclarationKindConstant,
			},
			{
				Name: "unknownType2",
				Type: sema.MetaType,
				Value: interpreter.TypeValue{
					Type: nil,
				},
				Kind: common.DeclarationKindConstant,
			},
		}

		semaValueDeclarations := valueDeclarations.ToSemaValueDeclarations()
		interpreterValueDeclarations := valueDeclarations.ToInterpreterValueDeclarations()

		inter := parseCheckAndInterpretWithOptions(t,
			`
              let result = unknownType1 == unknownType2
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(semaValueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(interpreterValueDeclarations),
				},
			},
		)

		assert.Equal(t,
			interpreter.BoolValue(false),
			inter.Globals["result"].Value,
		)
	})
}

func TestInterpretMetaTypeIdentifier(t *testing.T) {

	t.Parallel()

	t.Run("identifier, Int", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let type = Type<[Int]>()
          let identifier = type.identifier
        `)

		assert.Equal(t,
			interpreter.NewStringValue("[Int]"),
			inter.Globals["identifier"].Value,
		)
	})

	t.Run("identifier, struct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {}

          let type = Type<S>()
          let identifier = type.identifier
        `)

		assert.Equal(t,
			interpreter.NewStringValue("S.test.S"),
			inter.Globals["identifier"].Value,
		)
	})

	t.Run("unknown", func(t *testing.T) {

		t.Parallel()

		valueDeclarations := stdlib.StandardLibraryValues{
			{
				Name: "unknownType",
				Type: sema.MetaType,
				Value: interpreter.TypeValue{
					Type: nil,
				},
				Kind: common.DeclarationKindConstant,
			},
		}

		semaValueDeclarations := valueDeclarations.ToSemaValueDeclarations()
		interpreterValueDeclarations := valueDeclarations.ToInterpreterValueDeclarations()

		inter := parseCheckAndInterpretWithOptions(t,
			`
              let identifier = unknownType.identifier
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(semaValueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(interpreterValueDeclarations),
				},
			},
		)

		assert.Equal(t,
			interpreter.NewStringValue(""),
			inter.Globals["identifier"].Value,
		)
	})
}

func TestInterpretIsInstance(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name   string
		code   string
		result bool
	}{
		{
			name: "string is an instance of String",
			code: `
              let stringType = Type<String>()
              let result = "abc".isInstance(stringType)
            `,
			result: true,
		},
		{
			name: "int is an instance of Int",
			code: `
              let intType = Type<Int>()
              let result = (1).isInstance(intType)
            `,
			result: true,
		},
		{
			name: "resource is an instance of resource",
			code: `
              resource R {}

              let r <- create R()
              let rType = Type<@R>()
              let result = r.isInstance(rType)
            `,
			result: true,
		},
		{
			name: "int is not an instance of String",
			code: `
              let stringType = Type<String>()
              let result = (1).isInstance(stringType)
            `,
			result: false,
		},
		{
			name: "int is not an instance of resource",
			code: `
              resource R {}

              let rType = Type<@R>()
              let result = (1).isInstance(rType)
            `,
			result: false,
		},
		{
			name: "resource is not an instance of String",
			code: `
              resource R {}

              let r <- create R()
              let stringType = Type<String>()
              let result = r.isInstance(stringType)
            `,
			result: false,
		},
		{
			name: "resource R is not an instance of resource S",
			code: `
              resource R {}
              resource S {}

              let r <- create R()
              let sType = Type<@S>()
              let result = r.isInstance(sType)
            `,
			result: false,
		},
		{
			name: "struct S is not an instance of an unknown type",
			code: `
              struct S {}

              let s = S()
              let result = s.isInstance(unknownType)
            `,
			result: false,
		},
	}

	valueDeclarations := stdlib.StandardLibraryValues{
		{
			Name: "unknownType",
			Type: sema.MetaType,
			Value: interpreter.TypeValue{
				Type: nil,
			},
			Kind: common.DeclarationKindConstant,
		},
	}

	semaValueDeclarations := valueDeclarations.ToSemaValueDeclarations()
	interpreterValueDeclarations := valueDeclarations.ToInterpreterValueDeclarations()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			inter := parseCheckAndInterpretWithOptions(t, testCase.code, ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(semaValueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(interpreterValueDeclarations),
				},
			})

			assert.Equal(t,
				interpreter.BoolValue(testCase.result),
				inter.Globals["result"].Value,
			)
		})
	}
}

func TestInterpretGetType(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name   string
		code   string
		result interpreter.Value
	}{
		{
			name: "String",
			code: `
              let result = "abc".getType()
            `,
			result: interpreter.TypeValue{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			name: "Int",
			code: `
              let result = (1).getType()
            `,
			result: interpreter.TypeValue{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			name: "resource",
			code: `
              resource R {}

              let r <- create R()
              let result = r.getType()
            `,
			result: interpreter.TypeValue{
				Type: interpreter.CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "R",
				},
			},
		},
		{
			name: "array",
			code: `
              let result = [].getType()
            `,
			result: interpreter.TypeValue{
				// TODO: not yet supported
				Type: nil,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			inter := parseCheckAndInterpret(t, testCase.code)

			assert.Equal(t,
				testCase.result,
				inter.Globals["result"].Value,
			)
		})
	}
}
