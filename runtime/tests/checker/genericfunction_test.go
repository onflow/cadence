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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func parseAndCheckWithTestValue(t *testing.T, code string, ty sema.Type) (*sema.Checker, error) {

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
		Name: "test",
		Type: ty,
		Kind: common.DeclarationKindConstant,
	})

	return ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivation: baseValueActivation,
			},
		},
	)
}

func TestCheckGenericFunction(t *testing.T) {

	t.Parallel()

	t.Run("valid: no type parameters, no type arguments, no parameters, no arguments, no return type", func(t *testing.T) {

		for _, variant := range []string{"", "<>"} {

			checker, err := parseAndCheckWithTestValue(t,
				fmt.Sprintf(
					`
                      let res = test%s()
                    `,
					variant,
				),
				&sema.FunctionType{
					ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
				},
			)

			require.NoError(t, err)

			assert.Equal(t,
				sema.VoidType,
				RequireGlobalValue(t, checker.Elaboration, "res"),
			)
		}
	})

	t.Run("invalid: no type parameters, one type argument, no parameters, no arguments, no return type: too many type arguments", func(t *testing.T) {

		t.Parallel()

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test<X>()
            `,
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidTypeArgumentCountError{}, errs[0])
	})

	t.Run("invalid: one type parameter, no type argument, no parameters, no arguments: missing explicit type argument", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test()
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	t.Run("valid: one type parameter, one type argument, no parameters, no arguments", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		checker, err := parseAndCheckWithTestValue(t,
			`
              let res = test<Int>()
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		require.NoError(t, err)

		declarations := checker.Program.Declarations()

		require.IsType(t, &ast.VariableDeclaration{}, declarations[0])
		variableDeclaration := declarations[0].(*ast.VariableDeclaration)

		require.IsType(t, &ast.InvocationExpression{}, variableDeclaration.Value)
		invocationExpression := variableDeclaration.Value.(*ast.InvocationExpression)

		typeArguments := checker.Elaboration.InvocationExpressionTypes(invocationExpression).TypeArguments

		ty, present := typeArguments.Get(typeParameter)
		require.True(t, present, "could not find type argument for parameter %#+v", typeParameter)
		assert.IsType(t, sema.IntType, ty)
	})

	t.Run("valid: one type parameter, no type argument, one parameter, one arguments", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		checker, err := parseAndCheckWithTestValue(t,
			`
              let res = test(1)
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		require.NoError(t, err)

		declarations := checker.Program.Declarations()

		require.IsType(t, &ast.VariableDeclaration{}, declarations[0])
		variableDeclaration := declarations[0].(*ast.VariableDeclaration)

		require.IsType(t, &ast.InvocationExpression{}, variableDeclaration.Value)
		invocationExpression := variableDeclaration.Value.(*ast.InvocationExpression)

		typeArguments := checker.Elaboration.InvocationExpressionTypes(invocationExpression).TypeArguments

		ty, present := typeArguments.Get(typeParameter)
		require.True(t, present, "could not find type argument for type parameter %#+v", typeParameter)
		assert.IsType(t, sema.IntType, ty)
	})

	t.Run("invalid: one type parameter, no type argument, one parameter, no argument", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test()
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InsufficientArgumentsError{}, errs[0])
		assert.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
	})

	t.Run("invalid: one type parameter, one type argument, one parameter, one arguments: type mismatch", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test<Int>("1")
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("valid: one type parameter, one type argument, one parameter, one arguments", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test<Int>(1)
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		require.NoError(t, err)
	})

	t.Run("valid: one type parameter, no type argument, two parameters, two argument: matching argument types", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		checker, err := parseAndCheckWithTestValue(t,
			`
              let res = test(1, 2)
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "first",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "second",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		require.NoError(t, err)

		declarations := checker.Program.Declarations()

		require.IsType(t, &ast.VariableDeclaration{}, declarations[0])
		variableDeclaration := declarations[0].(*ast.VariableDeclaration)

		require.IsType(t, &ast.InvocationExpression{}, variableDeclaration.Value)
		invocationExpression := variableDeclaration.Value.(*ast.InvocationExpression)

		typeParameterTypes := checker.Elaboration.InvocationExpressionTypes(invocationExpression).TypeArguments

		ty, present := typeParameterTypes.Get(typeParameter)
		require.True(t, present, "could not find type argument for type parameter %#+v", typeParameter)
		assert.IsType(t, sema.IntType, ty)
	})

	t.Run("invalid: one type parameter, no type argument, two parameters, two argument: not matching argument types", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test(1, "2")
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "first",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "second",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("invalid: one type parameter, no type argument, no parameters, no arguments, return type", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test()
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	t.Run("valid: one type parameter, one type argument, no parameters, no arguments, return type", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		checker, err := parseAndCheckWithTestValue(t,
			`
              let res = test<Int>()
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
		)

		require.NoError(t, err)

		declarations := checker.Program.Declarations()

		require.IsType(t, &ast.VariableDeclaration{}, declarations[0])
		variableDeclaration := declarations[0].(*ast.VariableDeclaration)

		require.IsType(t, &ast.InvocationExpression{}, variableDeclaration.Value)
		invocationExpression := variableDeclaration.Value.(*ast.InvocationExpression)

		typeArguments := checker.Elaboration.InvocationExpressionTypes(invocationExpression).TypeArguments

		ty, present := typeArguments.Get(typeParameter)
		require.True(t, present, "could not find type argument for type parameter %#+v", typeParameter)
		assert.IsType(t, sema.IntType, ty)

		assert.IsType(t,
			sema.IntType,
			RequireGlobalValue(t, checker.Elaboration, "res"),
		)
	})

	t.Run("valid: one type parameter, one type argument, one parameter, one argument, return type", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: nil,
		}

		checker, err := parseAndCheckWithTestValue(t,
			`
              let res = test(1)
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
		)

		require.NoError(t, err)

		declarations := checker.Program.Declarations()

		require.IsType(t, &ast.VariableDeclaration{}, declarations[0])
		variableDeclaration := declarations[0].(*ast.VariableDeclaration)

		require.IsType(t, &ast.InvocationExpression{}, variableDeclaration.Value)
		invocationExpression := variableDeclaration.Value.(*ast.InvocationExpression)

		typeArguments := checker.Elaboration.InvocationExpressionTypes(invocationExpression).TypeArguments

		ty, present := typeArguments.Get(typeParameter)
		require.True(t, present, "could not find type argument for type parameter %#+v", typeParameter)
		assert.IsType(t, sema.IntType, ty)

		assert.IsType(t,
			sema.IntType,
			RequireGlobalValue(t, checker.Elaboration, "res"),
		)
	})

	t.Run("valid: one type parameter with type bound, one type argument, no parameters, no arguments", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: sema.NumberType,
		}

		checker, err := parseAndCheckWithTestValue(t,
			`
              let res = test<Int>()
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		require.NoError(t, err)

		declarations := checker.Program.Declarations()

		require.IsType(t, &ast.VariableDeclaration{}, declarations[0])
		variableDeclaration := declarations[0].(*ast.VariableDeclaration)

		require.IsType(t, &ast.InvocationExpression{}, variableDeclaration.Value)
		invocationExpression := variableDeclaration.Value.(*ast.InvocationExpression)

		typeArguments := checker.Elaboration.InvocationExpressionTypes(invocationExpression).TypeArguments

		ty, present := typeArguments.Get(typeParameter)
		require.True(t, present, "could not find type argument for type parameter %#+v", typeParameter)
		assert.IsType(t, sema.IntType, ty)
	})

	t.Run("invalid: one type parameter with type bound, one type argument, no parameters, no arguments: bound not satisfied", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: sema.NumberType,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test<String>()
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("invalid: one type parameter with type bound, no type argument, one parameter, one argument: bound not satisfied", func(t *testing.T) {

		t.Parallel()

		typeParameter := &sema.TypeParameter{
			Name:      "T",
			TypeBound: sema.NumberType,
		}

		_, err := parseAndCheckWithTestValue(t,
			`
              let res = test("test")
            `,
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter,
				},
				Parameters: []sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							&sema.GenericType{
								TypeParameter: typeParameter,
							},
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("valid: one type parameter, one type argument, no parameters, no arguments, generic return type", func(t *testing.T) {

		type test struct {
			name         string
			generateType func(innerType sema.Type) sema.Type
		}

		tests := []test{
			{
				name: "optional",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.OptionalType{
						Type: innerType,
					}
				},
			},
			{
				name: "variable-sized array",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.VariableSizedType{
						Type: innerType,
					}
				},
			},
			{
				name: "constant-sized array",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.ConstantSizedType{
						Type: innerType,
						Size: 2,
					}
				},
			},
			{
				name: "dictionary",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.DictionaryType{
						KeyType:   innerType,
						ValueType: innerType,
					}
				},
			},
		}

		for _, test := range tests {

			t.Run(test.name, func(t *testing.T) {

				typeParameter := &sema.TypeParameter{
					Name:      "T",
					TypeBound: sema.NumberType,
				}

				checker, err := parseAndCheckWithTestValue(t,
					`
                      let res = test<Int>()
                    `,
					&sema.FunctionType{
						TypeParameters: []*sema.TypeParameter{
							typeParameter,
						},
						Parameters: nil,
						ReturnTypeAnnotation: sema.NewTypeAnnotation(
							test.generateType(
								&sema.GenericType{
									TypeParameter: typeParameter,
								},
							),
						),
					},
				)

				require.NoError(t, err)

				assert.Equal(t,
					test.generateType(sema.IntType),
					RequireGlobalValue(t, checker.Elaboration, "res"),
				)
			})
		}
	})

	t.Run("valid: one type parameter, no type argument, one parameter, one argument, generic return type", func(t *testing.T) {

		type test struct {
			name         string
			generateType func(innerType sema.Type) sema.Type
			declarations string
			argument     string
		}

		tests := []test{
			{
				name: "optional",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.OptionalType{
						Type: innerType,
					}
				},
				declarations: "let x: Int? = 1",
				argument:     "x",
			},
			{
				name: "variable-sized array",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.VariableSizedType{
						Type: innerType,
					}
				},
				argument: "[1, 2, 3]",
			},
			{
				name: "constant-sized array",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.ConstantSizedType{
						Type: innerType,
						Size: 2,
					}
				},
				declarations: "let xs: [Int; 2] = [1, 2]",
				argument:     "xs",
			},
			{
				name: "dictionary",
				generateType: func(innerType sema.Type) sema.Type {
					return &sema.DictionaryType{
						KeyType:   innerType,
						ValueType: innerType,
					}
				},
				argument: "{1: 2}",
			},
		}

		for _, test := range tests {

			t.Run(test.name, func(t *testing.T) {

				typeParameter := &sema.TypeParameter{
					Name:      "T",
					TypeBound: sema.NumberType,
				}

				checker, err := parseAndCheckWithTestValue(t,
					fmt.Sprintf(
						`
                          %[1]s
                          let res = test(%[2]s)
                        `,
						test.declarations,
						test.argument,
					),
					&sema.FunctionType{
						TypeParameters: []*sema.TypeParameter{
							typeParameter,
						},
						Parameters: []sema.Parameter{
							{
								Label:      sema.ArgumentLabelNotRequired,
								Identifier: "value",
								TypeAnnotation: sema.NewTypeAnnotation(
									test.generateType(
										&sema.GenericType{
											TypeParameter: typeParameter,
										},
									),
								),
							},
						},
						ReturnTypeAnnotation: sema.NewTypeAnnotation(
							test.generateType(
								&sema.GenericType{
									TypeParameter: typeParameter,
								},
							),
						),
					},
				)

				require.NoError(t, err)

				assert.Equal(t,
					test.generateType(sema.IntType),
					RequireGlobalValue(t, checker.Elaboration, "res"),
				)
			})
		}
	})
}

// https://github.com/dapperlabs/flow-go/issues/3275
func TestCheckGenericFunctionIsInvalid(t *testing.T) {

	t.Parallel()

	typeParameter := &sema.TypeParameter{
		Name:      "T",
		TypeBound: nil,
	}

	genericFunctionType := &sema.FunctionType{
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
	}

	assert.False(t, genericFunctionType.IsInvalidType())
}

// https://github.com/onflow/cadence/issues/225
func TestCheckBorrowOfCapabilityWithoutTypeArgument(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t, `
      let cap: Capability = panic("")
      let ref = cap.borrow<&Int>()!
    `)

	require.NoError(t, err)
}

func TestCheckUnparameterizedTypeInstantiationE(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t, `
      struct S {}

      let s: S<Int> = panic("")
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UnparameterizedTypeInstantiationError{}, errs[0])
}
