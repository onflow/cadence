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
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/trampoline"
)

func TestInterpretVirtualImport(t *testing.T) {

	cryptoType := &sema.CompositeType{
		Location:   ast.IdentifierLocation("Crypto"),
		Identifier: "Crypto",
		Kind:       common.CompositeKindContract,
	}

	cryptoType.Members = map[string]*sema.Member{
		"unsafeRandom": sema.NewPublicFunctionMember(
			cryptoType,
			"unsafeRandom",
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(&sema.UInt64Type{}),
			},
		),
	}

	const code = `
       import Crypto

       fun test(): UInt64 {
           return Crypto.unsafeRandom()
       }
    `

	inter := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				interpreter.WithImportLocationHandler(
					func(inter *interpreter.Interpreter, location ast.Location) interpreter.Import {

						assert.Equal(t,
							ast.IdentifierLocation("Crypto"),
							location,
						)

						return interpreter.VirtualImport{
							Globals: map[string]interpreter.Value{
								"Crypto": &interpreter.CompositeValue{
									Location: location,
									TypeID:   "I.Crypto.Crypto",
									Kind:     common.CompositeKindContract,
									Functions: map[string]interpreter.FunctionValue{
										"unsafeRandom": interpreter.NewHostFunctionValue(
											func(invocation interpreter.Invocation) trampoline.Trampoline {
												return trampoline.Done{
													Result: interpreter.NewIntValueFromInt64(42),
												}
											},
										),
									},
								},
							},
						}
					},
				),
			},
			CheckerOptions: []sema.Option{
				sema.WithImportHandler(func(location ast.Location) sema.Import {
					return sema.VirtualImport{
						ValueElements: map[string]sema.ImportElement{
							"Crypto": {
								DeclarationKind: common.DeclarationKindStructure,
								Access:          ast.AccessPublic,
								Type:            cryptoType,
							},
						},
					}
				}),
			},
		},
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		value,
	)
}
