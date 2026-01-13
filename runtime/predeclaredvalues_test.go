/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package runtime_test

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestRuntimePredeclaredValues(t *testing.T) {

	t.Parallel()

	const contractName = "C"
	address := common.MustBytesToAddress([]byte{0x1})
	contractLocation := common.AddressLocation{
		Address: address,
		Name:    contractName,
	}

	test := func(
		t *testing.T,
		contract string,
		script string,
		valueDeclarations map[common.Location][]stdlib.StandardLibraryValue,
		typeDeclarations map[common.Location][]stdlib.StandardLibraryType,
		checkTransaction func(err error) bool,
		checkScript func(result cadence.Value, err error),
		useVM bool,
	) {

		runtime := NewTestRuntime()

		deploy := DeploymentTransaction(contractName, []byte(contract))

		var accountCode []byte
		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(_ Location) (bytes []byte, err error) {
				return accountCode, nil
			},
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
				return accountCode, nil
			},
			OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
				accountCode = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		prepareEnvironment := func(env Environment) {
			for location, valueDeclarations := range valueDeclarations {
				for _, valueDeclaration := range valueDeclarations {
					env.DeclareValue(valueDeclaration, location)
				}
			}
			for location, typeDeclarations := range typeDeclarations {
				for _, typeDeclaration := range typeDeclarations {
					env.DeclareType(typeDeclaration, location)
				}
			}
		}

		// Run deploy transaction

		transactionEnvironment := newTransactionEnvironment()
		prepareEnvironment(transactionEnvironment)

		err := runtime.ExecuteTransaction(
			Script{
				Source: deploy,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    common.TransactionLocation{},
				Environment: transactionEnvironment,
				UseVM:       *compile,
			},
		)

		// Run script if transaction was successful

		if checkTransaction(err) {

			var scriptEnvironment Environment
			if useVM {
				scriptEnvironment = NewScriptVMEnvironment(Config{})
			} else {
				scriptEnvironment = NewScriptInterpreterEnvironment(Config{})
			}
			prepareEnvironment(scriptEnvironment)

			checkScript(runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface:   runtimeInterface,
					Location:    common.ScriptLocation{},
					Environment: scriptEnvironment,
					UseVM:       useVM,
				},
			))
		}
	}

	t.Run("constant, everywhere", func(t *testing.T) {
		t.Parallel()

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return bar
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo() + bar
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				nil: {
					{
						Name:  "bar",
						Type:  sema.IntType,
						Kind:  common.DeclarationKindConstant,
						Value: interpreter.NewUnmeteredIntValueFromInt64(2),
					},
				},
			},
			nil,
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {

				require.NoError(t, err)

				require.Equal(t,
					cadence.Int{Value: big.NewInt(4)},
					result,
				)
			},
			*compile,
		)
	})

	t.Run("function, everywhere", func(t *testing.T) {
		t.Parallel()

		functionType := sema.NewSimpleFunctionType(
			sema.FunctionPurityView,
			nil,
			sema.IntTypeAnnotation,
		)

		nativeFunction := func(
			_ interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			_ interpreter.Value,
			_ []interpreter.Value,
		) interpreter.Value {
			return interpreter.NewUnmeteredIntValueFromInt64(2)
		}

		var function interpreter.FunctionValue
		if *compile {
			function = vm.NewNativeFunctionValue(
				"bar",
				functionType,
				nativeFunction,
			)
		} else {
			function = interpreter.NewStaticHostFunctionValueFromNativeFunction(
				nil,
				functionType,
				nativeFunction,
			)
		}

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return bar()
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo() + bar()
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				nil: {
					{
						Name:  "bar",
						Type:  functionType,
						Kind:  common.DeclarationKindFunction,
						Value: function,
					},
				},
			},
			nil,
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {

				require.NoError(t, err)

				require.Equal(t,
					cadence.Int{Value: big.NewInt(4)},
					result,
				)
			},
			*compile,
		)
	})

	t.Run("constant, only usable in contract, not used in script", func(t *testing.T) {
		t.Parallel()

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return bar
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo()
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				contractLocation: {
					{
						Name:  "bar",
						Type:  sema.IntType,
						Kind:  common.DeclarationKindConstant,
						Value: interpreter.NewUnmeteredIntValueFromInt64(2),
					},
				},
			},
			nil,
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {

				require.NoError(t, err)

				require.Equal(t,
					cadence.Int{Value: big.NewInt(2)},
					result,
				)
			},
			*compile,
		)
	})

	t.Run("constant, only usable in contract, used in script", func(t *testing.T) {
		t.Parallel()

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return bar
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return bar + C.foo()
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				contractLocation: {
					{
						Name:  "bar",
						Type:  sema.IntType,
						Kind:  common.DeclarationKindConstant,
						Value: interpreter.NewUnmeteredIntValueFromInt64(2),
					},
				},
			},
			nil,
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {
				RequireError(t, err)

				var checkerErr *sema.CheckerError
				require.ErrorAs(t, err, &checkerErr)
				assert.Equal(t, common.ScriptLocation{}, checkerErr.Location)

				errs := RequireCheckerErrors(t, err, 1)

				var notDeclaredErr *sema.NotDeclaredError
				require.ErrorAs(t, errs[0], &notDeclaredErr)
				require.Equal(t, "bar", notDeclaredErr.Name)
			},
			*compile,
		)
	})

	t.Run("function, only usable in contract, not used in script", func(t *testing.T) {
		t.Parallel()

		functionType := sema.NewSimpleFunctionType(
			sema.FunctionPurityView,
			nil,
			sema.IntTypeAnnotation,
		)

		var function interpreter.FunctionValue
		if *compile {
			function = vm.NewNativeFunctionValue(
				"bar",
				functionType,
				func(
					_ interpreter.NativeFunctionContext,
					_ interpreter.TypeArgumentsIterator,
					_ interpreter.ArgumentTypesIterator,
					_ interpreter.Value,
					_ []interpreter.Value,
				) interpreter.Value {
					return interpreter.NewUnmeteredIntValueFromInt64(2)
				},
			)
		} else {
			function = interpreter.NewStaticHostFunctionValue(
				nil,
				functionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					return interpreter.NewUnmeteredIntValueFromInt64(2)
				},
			)
		}

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return bar()
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo()
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				contractLocation: {
					{
						Name:  "bar",
						Type:  functionType,
						Kind:  common.DeclarationKindFunction,
						Value: function,
					},
				},
			},
			nil,
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {

				require.NoError(t, err)

				require.Equal(t,
					cadence.Int{Value: big.NewInt(2)},
					result,
				)
			},
			*compile,
		)
	})

	t.Run("function, only usable in contract, used in script", func(t *testing.T) {
		t.Parallel()

		functionType := sema.NewSimpleFunctionType(
			sema.FunctionPurityView,
			nil,
			sema.IntTypeAnnotation,
		)

		var function interpreter.FunctionValue
		if *compile {
			function = vm.NewNativeFunctionValue(
				"bar",
				functionType,
				func(
					_ interpreter.NativeFunctionContext,
					_ interpreter.TypeArgumentsIterator,
					_ interpreter.ArgumentTypesIterator,
					_ interpreter.Value,
					_ []interpreter.Value,
				) interpreter.Value {
					return interpreter.NewUnmeteredIntValueFromInt64(2)
				},
			)
		} else {
			function = interpreter.NewStaticHostFunctionValue(
				nil,
				functionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					return interpreter.NewUnmeteredIntValueFromInt64(2)
				},
			)
		}

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return bar()
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo() + bar()
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				contractLocation: {
					{
						Name:  "bar",
						Type:  functionType,
						Kind:  common.DeclarationKindFunction,
						Value: function,
					},
				},
			},
			nil,
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {
				RequireError(t, err)

				var checkerErr *sema.CheckerError
				require.ErrorAs(t, err, &checkerErr)
				assert.Equal(t, common.ScriptLocation{}, checkerErr.Location)

				errs := RequireCheckerErrors(t, err, 1)

				var notDeclaredErr *sema.NotDeclaredError
				require.ErrorAs(t, errs[0], &notDeclaredErr)
				require.Equal(t, "bar", notDeclaredErr.Name)
			},
			*compile,
		)
	})

	t.Run("contract, only usable in contract, not used in script", func(t *testing.T) {
		t.Parallel()

		cType := &sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "n",
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		}

		bType := &sema.CompositeType{
			Identifier: "B",
			Kind:       common.CompositeKindContract,
		}

		bType.Members = sema.MembersAsMap([]*sema.Member{
			sema.NewUnmeteredPublicFunctionMember(
				bType,
				"c",
				cType,
				"",
			),
			sema.NewUnmeteredPublicConstantFieldMember(
				bType,
				"d",
				sema.IntType,
				"",
			),
		})

		bStaticType := interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, bType)

		bValue := interpreter.NewSimpleCompositeValue(
			nil,
			bType.ID(),
			bStaticType,
			[]string{"d"},
			map[string]interpreter.Value{
				"d": interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			nil,
			nil,
			nil,
			nil,
		)

		var function interpreter.FunctionValue
		if *compile {
			function = vm.NewNativeFunctionValue(
				"B.c",
				cType,
				func(
					context interpreter.NativeFunctionContext,
					_ interpreter.TypeArgumentsIterator,
					_ interpreter.ArgumentTypesIterator,
					receiver interpreter.Value,
					args []interpreter.Value,
				) interpreter.Value {
					assert.Same(t, bValue, receiver)

					require.Len(t, args, 1)
					require.IsType(t, interpreter.IntValue{}, args[0])
					arg := args[0].(interpreter.IntValue)

					return arg.Plus(context, arg)
				},
			)
		} else {
			function = interpreter.NewStaticHostFunctionValue(
				nil,
				cType,
				func(invocation interpreter.Invocation) interpreter.Value {

					args := invocation.Arguments
					require.Len(t, args, 1)
					require.IsType(t, interpreter.IntValue{}, args[0])
					arg := args[0].(interpreter.IntValue)

					return arg.Plus(invocation.InvocationContext, arg)
				},
			)
			bValue.Fields["c"] = function
		}

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
                      return B.c(B.d)
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo()
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				contractLocation: {
					{
						Name:  bType.Identifier,
						Type:  bType,
						Value: bValue,
						Kind:  common.DeclarationKindContract,
					},
					// Only required for VM, not necessary for interpreter.
					{
						Name:  "B.c",
						Type:  cType,
						Value: function,
						Kind:  common.DeclarationKindFunction,
					},
				},
			},
			map[common.Location][]stdlib.StandardLibraryType{
				contractLocation: {
					{
						Name: bType.Identifier,
						Type: bType,
						Kind: common.DeclarationKindContract,
					},
				},
			},
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {

				require.NoError(t, err)

				require.Equal(t,
					cadence.Int{Value: big.NewInt(2)},
					result,
				)
			},
			*compile,
		)
	})

	t.Run("contract, only usable in contract, used in script", func(t *testing.T) {
		t.Parallel()

		cType := &sema.FunctionType{
			Parameters: []sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "n",
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		}

		bType := &sema.CompositeType{
			Identifier: "B",
			Kind:       common.CompositeKindContract,
		}

		bType.Members = sema.MembersAsMap([]*sema.Member{
			sema.NewUnmeteredPublicFunctionMember(
				bType,
				"c",
				cType,
				"",
			),
			sema.NewUnmeteredPublicConstantFieldMember(
				bType,
				"d",
				sema.IntType,
				"",
			),
		})

		var function interpreter.FunctionValue
		if *compile {
			function = vm.NewNativeFunctionValue(
				"B.c",
				cType,
				func(
					_ interpreter.NativeFunctionContext,
					_ interpreter.TypeArgumentsIterator,
					_ interpreter.ArgumentTypesIterator,
					_ interpreter.Value,
					_ []interpreter.Value,
				) interpreter.Value {
					require.Fail(t, "function should have not been called")
					return nil
				},
			)
		} else {
			function = interpreter.NewStaticHostFunctionValue(
				nil,
				cType,
				func(invocation interpreter.Invocation) interpreter.Value {
					require.Fail(t, "function should have not been called")
					return nil
				},
			)
		}

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
                      return B.c(B.d)
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo() + B.c(B.d)
	          }
	        `,
			map[common.Location][]stdlib.StandardLibraryValue{
				contractLocation: {
					{
						Name: bType.Identifier,
						Type: bType,
						// Omitted, not used
						Kind: common.DeclarationKindContract,
					},
					// Only required for VM, not necessary for interpreter.
					{
						Name:  "B.c",
						Type:  cType,
						Value: function,
						Kind:  common.DeclarationKindFunction,
					},
				},
			},
			map[common.Location][]stdlib.StandardLibraryType{
				contractLocation: {
					{
						Name: bType.Identifier,
						Type: bType,
						Kind: common.DeclarationKindContract,
					},
				},
			},
			func(err error) bool {
				return assert.NoError(t, err)
			},
			func(result cadence.Value, err error) {
				RequireError(t, err)

				var checkerErr *sema.CheckerError
				require.ErrorAs(t, err, &checkerErr)
				assert.Equal(t, common.ScriptLocation{}, checkerErr.Location)

				errs := RequireCheckerErrors(t, err, 2)

				var notDeclaredErr *sema.NotDeclaredError
				require.ErrorAs(t, errs[0], &notDeclaredErr)
				assert.Equal(t, "B", notDeclaredErr.Name)

				require.ErrorAs(t, errs[1], &notDeclaredErr)
				assert.Equal(t, "B", notDeclaredErr.Name)
			},
			*compile,
		)
	})

}

func TestRuntimePredeclaredTypes(t *testing.T) {

	t.Parallel()

	t.Run("type alias", func(t *testing.T) {
		t.Parallel()

		xType := sema.IntType

		valueDeclaration := stdlib.StandardLibraryValue{
			Name:  "x",
			Type:  xType,
			Kind:  common.DeclarationKindConstant,
			Value: interpreter.NewUnmeteredIntValueFromInt64(2),
		}

		typeDeclaration := stdlib.StandardLibraryType{
			Name: "X",
			Type: xType,
			Kind: common.DeclarationKindType,
		}

		script := []byte(`
          access(all)
          fun main(): X {
              return x
          }
	    `)

		runtime := NewTestRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := newScriptEnvironment()
		scriptEnvironment.DeclareValue(valueDeclaration, nil)
		scriptEnvironment.DeclareType(typeDeclaration, nil)

		result, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    common.ScriptLocation{},
				Environment: scriptEnvironment,
				UseVM:       *compile,
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			cadence.Int{Value: big.NewInt(2)},
			result,
		)
	})

	t.Run("composite type, top-level, existing", func(t *testing.T) {
		t.Parallel()

		xType := &sema.CompositeType{
			Identifier: "X",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
		}

		valueDeclaration := stdlib.StandardLibraryValue{
			Name: "x",
			Type: xType,
			Kind: common.DeclarationKindConstant,
			Value: interpreter.NewSimpleCompositeValue(nil,
				xType.ID(),
				interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, xType),
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
		}

		typeDeclaration := stdlib.StandardLibraryType{
			Name: "X",
			Type: xType,
			Kind: common.DeclarationKindType,
		}

		script := []byte(`
          access(all)
          fun main(): X {
              return x
          }
	    `)

		runtime := NewTestRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := newScriptEnvironment()
		scriptEnvironment.DeclareValue(valueDeclaration, nil)
		scriptEnvironment.DeclareType(typeDeclaration, nil)

		result, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    common.ScriptLocation{},
				Environment: scriptEnvironment,
				UseVM:       *compile,
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			cadence.NewStruct([]cadence.Value{}).
				WithType(cadence.NewStructType(
					nil,
					xType.QualifiedIdentifier(),
					[]cadence.Field{},
					nil,
				)),
			result,
		)
	})

	t.Run("composite type, top-level, non-existing", func(t *testing.T) {
		t.Parallel()

		location := common.ScriptLocation{}

		xType := &sema.CompositeType{
			Location:   location,
			Identifier: "X",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
		}

		valueDeclaration := stdlib.StandardLibraryValue{
			Name: "x",
			Type: xType,
			Kind: common.DeclarationKindConstant,
			Value: interpreter.NewSimpleCompositeValue(nil,
				xType.ID(),
				interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, xType),
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
		}

		script := []byte(`
          access(all)
          fun main(): AnyStruct {
              return x
          }
	    `)

		runtime := NewTestRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := newScriptEnvironment()
		scriptEnvironment.DeclareValue(valueDeclaration, nil)

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    location,
				Environment: scriptEnvironment,
				UseVM:       *compile,
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})

	t.Run("composite type, nested, existing", func(t *testing.T) {
		t.Parallel()

		location := common.ScriptLocation{}

		yType := &sema.CompositeType{
			Location:   location,
			Identifier: "Y",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
		}

		xType := &sema.CompositeType{
			Location:   location,
			Identifier: "X",
			Kind:       common.CompositeKindContract,
			Members:    &sema.StringMemberOrderedMap{},
		}

		xType.SetNestedType(yType.Identifier, yType)

		valueDeclaration := stdlib.StandardLibraryValue{
			Name: "y",
			Type: yType,
			Kind: common.DeclarationKindConstant,
			Value: interpreter.NewSimpleCompositeValue(nil,
				yType.ID(),
				interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, yType),
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
		}

		xTypeDeclaration := stdlib.StandardLibraryType{
			Name: "X",
			Type: xType,
			Kind: common.DeclarationKindType,
		}

		yTypeDeclaration := stdlib.StandardLibraryType{
			Name: "X.Y",
			Type: yType,
			Kind: common.DeclarationKindType,
		}

		script := []byte(`
          access(all)
          fun main(): X.Y {
              return y
          }
	    `)

		runtime := NewTestRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := newScriptEnvironment()
		scriptEnvironment.DeclareValue(valueDeclaration, nil)
		scriptEnvironment.DeclareType(xTypeDeclaration, nil)
		scriptEnvironment.DeclareType(yTypeDeclaration, nil)

		result, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    location,
				Environment: scriptEnvironment,
				UseVM:       *compile,
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			cadence.NewStruct([]cadence.Value{}).
				WithType(cadence.NewStructType(
					location,
					yType.QualifiedIdentifier(),
					[]cadence.Field{},
					nil,
				)),
			result,
		)
	})

	t.Run("composite type, nested, non-existing", func(t *testing.T) {
		t.Parallel()

		location := common.ScriptLocation{}

		yType := &sema.CompositeType{
			Location:   location,
			Identifier: "Y",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
		}

		xType := &sema.CompositeType{
			Location:   location,
			Identifier: "X",
			Kind:       common.CompositeKindContract,
			Members:    &sema.StringMemberOrderedMap{},
		}

		xType.SetNestedType(yType.Identifier, yType)

		valueDeclaration := stdlib.StandardLibraryValue{
			Name: "y",
			Type: yType,
			Kind: common.DeclarationKindConstant,
			Value: interpreter.NewSimpleCompositeValue(nil,
				yType.ID(),
				interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, yType),
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
		}

		script := []byte(`
          access(all)
          fun main(): AnyStruct {
              return y
          }
	    `)

		runtime := NewTestRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := newScriptEnvironment()
		scriptEnvironment.DeclareValue(valueDeclaration, nil)

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    location,
				Environment: scriptEnvironment,
				UseVM:       *compile,
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})

}

// NOTE: This feature is only supported by the interpreter environment, not the VM environment.
func TestRuntimePredeclaredTypeWithInjectedFunctions(t *testing.T) {

	t.Parallel()

	xType := &sema.CompositeType{
		Identifier: "X",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}

	const fooFunctionName = "foo"
	fooFunctionType := &sema.FunctionType{
		Parameters: []sema.Parameter{
			{
				Identifier:     "bar",
				TypeAnnotation: sema.NewTypeAnnotation(sema.UInt8Type),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
	}

	fooFunctionMember := sema.NewPublicFunctionMember(
		nil,
		xType,
		fooFunctionName,
		fooFunctionType,
		"",
	)
	xType.Members.Set(fooFunctionName, fooFunctionMember)

	xConstructorType := &sema.FunctionType{
		ReturnTypeAnnotation: sema.NewTypeAnnotation(xType),
	}

	xConstructorDeclaration := stdlib.StandardLibraryValue{
		Name: "X",
		Type: xConstructorType,
		Kind: common.DeclarationKindConstant,
		Value: interpreter.NewStaticHostFunctionValue(
			nil,
			xConstructorType,
			func(invocation interpreter.Invocation) interpreter.Value {
				return interpreter.NewCompositeValue(
					invocation.InvocationContext,
					xType.Location,
					xType.QualifiedIdentifier(),
					xType.Kind,
					nil,
					common.ZeroAddress,
				)
			},
		),
	}

	xTypeDeclaration := stdlib.StandardLibraryType{
		Name: "X",
		Type: xType,
		Kind: common.DeclarationKindType,
	}

	script := []byte(`
      access(all)
      fun main(): String {
          return X().foo(bar: 1)
      }
	`)

	runtime := NewTestRuntime()

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
	}

	// Run script

	scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
	scriptEnvironment.DeclareValue(xConstructorDeclaration, nil)
	scriptEnvironment.DeclareType(xTypeDeclaration, nil)
	scriptEnvironment.SetCompositeValueFunctionsHandler(
		xType.ID(),
		func(
			inter *interpreter.Interpreter,
			compositeValue *interpreter.CompositeValue,
		) *interpreter.FunctionOrderedMap {
			require.NotNil(t, compositeValue)

			functions := orderedmap.New[interpreter.FunctionOrderedMap](1)
			functions.Set(fooFunctionName, interpreter.NewStaticHostFunctionValue(
				inter,
				fooFunctionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					arg := invocation.Arguments[0]
					require.IsType(t, interpreter.UInt8Value(0), arg)

					return interpreter.NewUnmeteredStringValue(strconv.Itoa(int(arg.(interpreter.UInt8Value) + 1)))
				},
			))
			return functions
		},
	)

	result, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface:   runtimeInterface,
			Location:    common.ScriptLocation{},
			Environment: scriptEnvironment,
			// NOTE: not supported by VM environment
			UseVM: false,
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		cadence.String("2"),
		result,
	)

}

func TestRuntimeArgumentTypes(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	functionType := sema.NewSimpleFunctionType(
		sema.FunctionPurityView,
		[]sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: sema.AnyStructTypeAnnotation,
			},
		},
		sema.VoidTypeAnnotation,
	)
	functionType.Arity = &sema.Arity{Min: 1}

	var called bool
	checkArgumentTypes := func(argumentTypes interpreter.ArgumentTypesIterator) {
		called = true

		argType := argumentTypes.NextSema()
		assert.Equal(t, sema.IntType, argType)

		argType = argumentTypes.NextSema()
		assert.Equal(t, sema.StringType, argType)

		assert.Nil(t, argumentTypes.NextSema())
	}

	const functionName = "foo"

	var function interpreter.FunctionValue
	if *compile {
		function = vm.NewNativeFunctionValue(
			functionName,
			functionType,
			func(
				_ interpreter.NativeFunctionContext,
				_ interpreter.TypeArgumentsIterator,
				argumentTypes interpreter.ArgumentTypesIterator,
				_ interpreter.Value,
				_ []interpreter.Value,
			) interpreter.Value {
				checkArgumentTypes(argumentTypes)
				return interpreter.Void
			},
		)
	} else {
		function = interpreter.NewStaticHostFunctionValue(
			nil,
			functionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				argumentTypes := interpreter.NewArgumentTypesIterator(
					invocation.InvocationContext,
					invocation.ArgumentTypes,
				)
				checkArgumentTypes(argumentTypes)
				return interpreter.Void
			},
		)
	}

	env := newTransactionEnvironment()

	env.DeclareValue(
		stdlib.StandardLibraryValue{
			Name:  functionName,
			Type:  functionType,
			Kind:  common.DeclarationKindFunction,
			Value: function,
		},
		nil,
	)

	tx := `
      transaction {
          prepare(signer: &Account) {
              foo(42, "forty two")
          }
      }
    `

	runtime := NewTestRuntime()

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(tx),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    common.TransactionLocation{},
			Environment: env,
			UseVM:       *compile,
		},
	)
	require.NoError(t, err)

	assert.True(t, called)
}
