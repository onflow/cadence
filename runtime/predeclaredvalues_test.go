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

package runtime_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
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
	  access(all) contract C {
	      access(all) fun foo(): Int {
	          return foo
	      }
	  }
	`)

	script := []byte(`
	  import C from 0x1

	  access(all) fun main(): Int {
		  return foo + C.foo()
	  }
	`)

	runtime := NewTestInterpreterRuntime()

	deploy := DeploymentTransaction("C", contract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
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
			cadence.Struct{
				StructType: cadence.NewStructType(nil, xType.QualifiedIdentifier(), []cadence.Field{}, nil),
				Fields:     []cadence.Value{},
			},
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
			cadence.Struct{
				StructType: cadence.NewStructType(nil, yType.QualifiedIdentifier(), []cadence.Field{}, nil),
				Fields:     []cadence.Value{},
			},
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
