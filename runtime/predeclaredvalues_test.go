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
		valueDeclarations map[common.Location]stdlib.StandardLibraryValue,
		checkTransaction func(err error) bool,
		checkScript func(result cadence.Value, err error),
	) {

		runtime := NewTestInterpreterRuntime()

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
			for location, valueDeclaration := range valueDeclarations {
				env.DeclareValue(valueDeclaration, location)
			}
		}

		// Run deploy transaction

		transactionEnvironment := NewBaseInterpreterEnvironment(Config{})
		prepareEnvironment(transactionEnvironment)

		err := runtime.ExecuteTransaction(
			Script{
				Source: deploy,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    common.TransactionLocation{},
				Environment: transactionEnvironment,
			},
		)

		if checkTransaction(err) {

			scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
			prepareEnvironment(scriptEnvironment)

			checkScript(runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface:   runtimeInterface,
					Location:    common.ScriptLocation{},
					Environment: scriptEnvironment,
				},
			))
		}
	}

	t.Run("everywhere", func(t *testing.T) {
		t.Parallel()

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return foo
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return foo + C.foo()
	          }
	        `,
			map[common.Location]stdlib.StandardLibraryValue{
				nil: {
					Name:  "foo",
					Type:  sema.IntType,
					Kind:  common.DeclarationKindConstant,
					Value: interpreter.NewUnmeteredIntValueFromInt64(2),
				},
			},
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
		)
	})

	t.Run("only contract, no use in script", func(t *testing.T) {
		t.Parallel()

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return foo
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return C.foo()
	          }
	        `,
			map[common.Location]stdlib.StandardLibraryValue{
				contractLocation: {
					Name:  "foo",
					Type:  sema.IntType,
					Kind:  common.DeclarationKindConstant,
					Value: interpreter.NewUnmeteredIntValueFromInt64(2),
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
		)
	})

	t.Run("only contract, use in script", func(t *testing.T) {
		t.Parallel()

		test(t,
			`
	          access(all) contract C {
	              access(all) fun foo(): Int {
	                  return foo
	              }
	          }
	        `,
			`
	          import C from 0x1

	          access(all) fun main(): Int {
	        	  return foo + C.foo()
	          }
	        `,
			map[common.Location]stdlib.StandardLibraryValue{
				contractLocation: {
					Name:  "foo",
					Type:  sema.IntType,
					Kind:  common.DeclarationKindConstant,
					Value: interpreter.NewUnmeteredIntValueFromInt64(2),
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

				errs := RequireCheckerErrors(t, err, 1)

				var notDeclaredErr *sema.NotDeclaredError
				require.ErrorAs(t, errs[0], &notDeclaredErr)
				require.Equal(t, "foo", notDeclaredErr.Name)
			},
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

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
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

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
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
			),
		}

		script := []byte(`
          access(all)
          fun main(): AnyStruct {
              return x
          }
	    `)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
		scriptEnvironment.DeclareValue(valueDeclaration, nil)

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    common.ScriptLocation{},
				Environment: scriptEnvironment,
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})

	t.Run("composite type, nested, existing", func(t *testing.T) {
		t.Parallel()

		yType := &sema.CompositeType{
			Identifier: "Y",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
		}

		xType := &sema.CompositeType{
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
			),
		}

		typeDeclaration := stdlib.StandardLibraryType{
			Name: "X",
			Type: xType,
			Kind: common.DeclarationKindType,
		}

		script := []byte(`
          access(all)
          fun main(): X.Y {
              return y
          }
	    `)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
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
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			cadence.NewStruct([]cadence.Value{}).
				WithType(cadence.NewStructType(
					nil,
					yType.QualifiedIdentifier(),
					[]cadence.Field{},
					nil,
				)),
			result,
		)
	})

	t.Run("composite type, nested, non-existing", func(t *testing.T) {
		t.Parallel()

		yType := &sema.CompositeType{
			Identifier: "Y",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
		}

		xType := &sema.CompositeType{
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
			),
		}

		script := []byte(`
          access(all)
          fun main(): AnyStruct {
              return y
          }
	    `)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
		scriptEnvironment.DeclareValue(valueDeclaration, nil)

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    common.ScriptLocation{},
				Environment: scriptEnvironment,
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})

}

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
					invocation.Interpreter,
					invocation.LocationRange,
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

	runtime := NewTestInterpreterRuntime()

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
			locationRange interpreter.LocationRange,
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
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		cadence.String("2"),
		result,
	)

}
