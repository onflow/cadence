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

package runtime

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimePredeclaredValues(t *testing.T) {

	t.Parallel()

	valueDeclaration := stdlib.StandardLibraryValue{
		Name:  "foo",
		Type:  sema.IntType,
		Kind:  common.DeclarationKindConstant,
		Value: interpreter.NewUnmeteredIntValueFromInt64(2),
	}

	contract := []byte(`
	  pub contract C {
	      pub fun foo(): Int {
	          return foo
	      }
	  }
	`)

	script := []byte(`
	  import C from 0x1

	  pub fun main(): Int {
		  return foo + C.foo()
	  }
	`)

	runtime := newTestInterpreterRuntime()

	deploy := DeploymentTransaction("C", contract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	// Run transaction

	transactionEnvironment := NewBaseInterpreterEnvironment(Config{})
	transactionEnvironment.DeclareValue(valueDeclaration)

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
	require.NoError(t, err)

	// Run script

	scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
	scriptEnvironment.DeclareValue(valueDeclaration)

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
		cadence.Int{Value: big.NewInt(4)},
		result,
	)
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
          pub fun main(): X {
              return x
          }
	    `)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
		scriptEnvironment.DeclareValue(valueDeclaration)
		scriptEnvironment.DeclareType(typeDeclaration)

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
          pub fun main(): X {
              return x
          }
	    `)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
		scriptEnvironment.DeclareValue(valueDeclaration)
		scriptEnvironment.DeclareType(typeDeclaration)

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
			cadence.ValueWithCachedTypeID(
				cadence.Struct{
					StructType: cadence.NewStructType(nil, xType.QualifiedIdentifier(), []cadence.Field{}, nil),
					Fields:     []cadence.Value{},
				},
			),
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
          pub fun main(): AnyStruct {
              return x
          }
	    `)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
		scriptEnvironment.DeclareValue(valueDeclaration)

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
          pub fun main(): X.Y {
              return y
          }
	    `)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
		scriptEnvironment.DeclareValue(valueDeclaration)
		scriptEnvironment.DeclareType(typeDeclaration)

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
			cadence.ValueWithCachedTypeID(
				cadence.Struct{
					StructType: cadence.NewStructType(nil, yType.QualifiedIdentifier(), []cadence.Field{}, nil),
					Fields:     []cadence.Value{},
				},
			),
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
          pub fun main(): AnyStruct {
              return y
          }
	    `)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
		}

		// Run script

		scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
		scriptEnvironment.DeclareValue(valueDeclaration)

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
		Value: interpreter.NewHostFunctionValue(
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
      pub fun main(): String {
          return X().foo(bar: 1)
      }
	`)

	runtime := newTestInterpreterRuntime()

	runtimeInterface := &testRuntimeInterface{
		storage:         newTestLedger(nil, nil),
		resolveLocation: singleIdentifierLocationResolver(t),
	}

	// Run script

	scriptEnvironment := NewScriptInterpreterEnvironment(Config{})
	scriptEnvironment.DeclareValue(xConstructorDeclaration)
	scriptEnvironment.DeclareType(xTypeDeclaration)
	scriptEnvironment.SetCompositeValueFunctionsHandler(
		xType.ID(),
		func(
			inter *interpreter.Interpreter,
			locationRange interpreter.LocationRange,
			compositeValue *interpreter.CompositeValue,
		) map[string]interpreter.FunctionValue {
			require.NotNil(t, compositeValue)

			return map[string]interpreter.FunctionValue{
				fooFunctionName: interpreter.NewHostFunctionValue(
					inter,
					fooFunctionType,
					func(invocation interpreter.Invocation) interpreter.Value {
						arg := invocation.Arguments[0]
						require.IsType(t, interpreter.UInt8Value(0), arg)

						return interpreter.NewUnmeteredStringValue(strconv.Itoa(int(arg.(interpreter.UInt8Value) + 1)))
					},
				),
			}
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

func TestRuntimePredeclaredStorableType(t *testing.T) {

	t.Parallel()

	xType := &sema.CompositeType{
		Identifier:      "X",
		Kind:            common.CompositeKindStructure,
		Members:         &sema.StringMemberOrderedMap{},
		StorableBuiltin: true,
	}

	const fooFieldName = "foo"

	fooFieldMember := sema.NewPublicConstantFieldMember(
		nil,
		xType,
		fooFieldName,
		sema.IntType,
		"",
	)
	xType.Members.Set(fooFieldName, fooFieldMember)

	xConstructorType := &sema.FunctionType{
		ReturnTypeAnnotation: sema.NewTypeAnnotation(xType),
	}

	xConstructorDeclaration := stdlib.StandardLibraryValue{
		Name: "X",
		Type: xConstructorType,
		Kind: common.DeclarationKindConstant,
		Value: interpreter.NewHostFunctionValue(
			nil,
			xConstructorType,
			func(invocation interpreter.Invocation) interpreter.Value {
				return interpreter.NewCompositeValue(
					invocation.Interpreter,
					invocation.LocationRange,
					xType.Location,
					xType.QualifiedIdentifier(),
					xType.Kind,
					[]interpreter.CompositeField{
						{
							Name:  fooFieldName,
							Value: interpreter.NewUnmeteredIntValueFromInt64(42),
						},
					},
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

	createAndSaveTx := []byte(`
      transaction {
          prepare(signer: AuthAccount) {
             signer.save(X(), to: /storage/x)
          }
      }
	`)

	loadAndUseTx := []byte(`
	 transaction {
	     prepare(signer: AuthAccount) {
	        let x = signer.load<X>(from: /storage/x)!
	        log(x.foo)
	     }
	 }
	`)

	runtime := newTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	var logs []string

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		log: func(message string) {
			logs = append(logs, message)
		},
	}

	// Create environment

	txEnvironment := NewBaseInterpreterEnvironment(Config{})
	txEnvironment.DeclareValue(xConstructorDeclaration)
	txEnvironment.DeclareType(xTypeDeclaration)

	nextTransactionLocation := newTransactionLocationGenerator()

	// Create and save

	err := runtime.ExecuteTransaction(
		Script{
			Source: createAndSaveTx,
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: txEnvironment,
		},
	)
	require.NoError(t, err)

	// Load and use

	err = runtime.ExecuteTransaction(
		Script{
			Source: loadAndUseTx,
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: txEnvironment,
		},
	)
	require.NoError(t, err)

	require.Equal(t, []string{"42"}, logs)
}
