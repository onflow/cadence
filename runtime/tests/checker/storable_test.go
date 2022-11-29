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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckStorable(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string, errorTypes ...error) {

		_, err := ParseAndCheckWithPanic(t, code)

		if len(errorTypes) == 0 {
			require.NoError(t, err)
		} else {
			errs := RequireCheckerErrors(t, err, len(errorTypes))
			for i, errorType := range errorTypes {
				require.IsType(t, errorType, errs[i])
			}
		}
	}

	generateNestedTypes := func(ty sema.Type, includeCapability bool) []sema.Type {

		// TODO: Restricted

		nestedTypes := []sema.Type{
			&sema.OptionalType{Type: ty},
			&sema.VariableSizedType{Type: ty},
			&sema.ConstantSizedType{Type: ty, Size: 1},
			&sema.DictionaryType{
				KeyType:   sema.BoolType,
				ValueType: ty,
			},
		}

		if includeCapability {
			nestedTypes = append(nestedTypes,
				&sema.CapabilityType{
					BorrowType: &sema.ReferenceType{
						Type: ty,
					},
				},
			)
		}

		if sema.IsValidDictionaryKeyType(ty) {
			nestedTypes = append(nestedTypes,
				&sema.DictionaryType{
					KeyType:   ty,
					ValueType: sema.BoolType,
				},
			)
		}

		return nestedTypes
	}

	type testCase struct {
		Type       sema.Type
		TypeName   string
		ErrorTypes func(compositeKind common.CompositeKind, isInterface bool) []error
	}

	var testCases []testCase

	storableTypes := sema.AllNumberTypes[:]
	storableTypes = append(
		storableTypes,
		sema.TheAddressType,
		sema.PathType,
		&sema.CapabilityType{},
		sema.StringType,
		sema.BoolType,
		sema.MetaType,
		sema.CharacterType,
		sema.AnyStructType,
		sema.AnyResourceType,
	)

	for _, storableType := range storableTypes {
		storableTypes = append(
			storableTypes,
			generateNestedTypes(storableType, true)...,
		)
	}

	nonStorableTypes := []sema.Type{
		&sema.FunctionType{
			ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		},
		sema.NeverType,
		sema.VoidType,
		sema.AuthAccountType,
		sema.PublicAccountType,
	}

	// Capabilities of non-storable types are storable

	for _, nonStorableType := range nonStorableTypes {
		storableTypes = append(
			storableTypes,
			&sema.CapabilityType{
				BorrowType: &sema.ReferenceType{
					Type: nonStorableType,
				},
			},
		)
	}

	nonStorableTypes = append(nonStorableTypes,
		&sema.ReferenceType{
			Type: sema.BoolType,
		},
	)

	for _, storableType := range storableTypes {
		testCases = append(testCases,
			testCase{
				Type:       storableType,
				ErrorTypes: nil,
			},
		)
	}

	// Invalid types are storable

	testCases = append(testCases,
		testCase{
			TypeName: "NotDeclaredType",
			ErrorTypes: func(compositeKind common.CompositeKind, isInterface bool) []error {
				if isInterface || compositeKind == common.CompositeKindEvent {
					return []error{
						&sema.NotDeclaredError{},
					}
				} else {
					return []error{
						&sema.NotDeclaredError{},
						&sema.NotDeclaredError{},
					}
				}
			},
		},
	)

	for _, nonStorableType := range nonStorableTypes[:] {
		nonStorableTypes = append(
			nonStorableTypes,
			generateNestedTypes(nonStorableType, false)...,
		)
	}

	for _, nonStorableType := range nonStorableTypes {
		testCases = append(testCases,
			testCase{
				Type: nonStorableType,
				ErrorTypes: func(kind common.CompositeKind, _ bool) []error {
					if kind == common.CompositeKindContract {
						return []error{
							&sema.FieldTypeNotStorableError{},
						}
					}

					return nil
				},
			},
		)
	}

	// Check all test cases

	for _, testCase := range testCases {

		// Generate an check composite or interface, for all composite kinds,
		// with a field of the test case's type

		// Determine the type name and type annotation

		typeAnnotation := ""
		typeName := testCase.TypeName
		isResource := false

		if testCase.Type != nil {
			isResource = testCase.Type.IsResourceType()
			if isResource {
				typeAnnotation = "@"
			}
			typeName = testCase.Type.String()
		}

		for _, compositeKind := range common.AllCompositeKinds {

			// Skip enums, they can't have fields

			if compositeKind == common.CompositeKindEnum {
				continue
			}

			// Skip types that cannot be used in events

			if compositeKind == common.CompositeKindEvent &&
				testCase.Type != nil &&
				!sema.IsValidEventParameterType(testCase.Type, map[*sema.Member]bool{}) {

				continue
			}

			// Skip resource types in non-resource composites

			if isResource && compositeKind != common.CompositeKindResource {
				continue
			}

			for _, isInterface := range []bool{true, false} {

				// Skip composite kinds that don't support interfaces

				if isInterface && !compositeKind.SupportsInterfaces() {
					continue
				}

				var errorTypes []error
				if testCase.ErrorTypes != nil {
					errorTypes = testCase.ErrorTypes(compositeKind, isInterface)
				}

				var interfaceKeyword string
				var initializer string
				var destructor string

				if isInterface {
					interfaceKeyword = "interface"
				} else {

					transferOperation := ast.TransferOperationCopy

					if testCase.Type != nil && testCase.Type.IsResourceType() {
						transferOperation = ast.TransferOperationMove
					}

					// In composite declarations (non-interface declarations),
					// the field needs an initializer.
					//
					// If the tested type is a resource, it also needs a destructor.

					initializer = fmt.Sprintf(
						`
                              init(value: %[1]s%[2]s) {
                                  self.value %[3]s value
                              }
                        `,
						typeAnnotation,
						typeName,
						transferOperation.Operator(),
					)

					if isResource {
						destructor = `
                              destroy() {
                                  destroy self.value
                              }
                        `
					}
				}

				var body string
				if compositeKind == common.CompositeKindEvent {
					body = fmt.Sprintf("(value: %s)", typeName)
				} else {
					body = fmt.Sprintf(
						`{
                              let value: %[1]s%[2]s

                              %[3]s

                              %[4]s
                          }
                        `,
						typeAnnotation,
						typeName,
						initializer,
						destructor,
					)
				}

				compositeKeyword := compositeKind.Keyword()
				testName := fmt.Sprintf(
					"%s in %s %s",
					typeName,
					compositeKeyword,
					interfaceKeyword,
				)

				t.Run(testName, func(t *testing.T) {

					t.Parallel()

					code := fmt.Sprintf(
						`
					      %[1]s %[2]s T %[3]s
					    `,
						compositeKeyword,
						interfaceKeyword,
						body,
					)

					test(t,
						code,
						errorTypes...,
					)
				})
			}
		}
	}
}

func TestCheckStorableCycle(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          struct S {
              let s: S

              init() {
                  self.s = S()
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("nested with same name", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
           struct S {

               struct S {
                   let s: S
               }

               let s: S
           }
        `)

		errs := RequireCheckerErrors(t, err, 5)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.RedeclarationError{}, errs[1])
		assert.IsType(t, &sema.RedeclarationError{}, errs[2])
		assert.IsType(t, &sema.MissingInitializerError{}, errs[3])
		assert.IsType(t, &sema.MissingInitializerError{}, errs[4])
	})
}
