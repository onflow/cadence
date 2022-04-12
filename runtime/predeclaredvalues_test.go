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

package runtime

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
)

func TestRuntimePredeclaredValues(t *testing.T) {

	t.Parallel()

	// Declare four programs.
	// Program 0x1 imports 0x2, 0x3, and 0x4.
	// All programs attempt to call a function 'foo'.
	// Only predeclare a function 'foo' for 0x2 and 0x4.
	// Both functions have the same name, but different types.

	address2 := common.MustBytesToAddress([]byte{0x2})
	address3 := common.MustBytesToAddress([]byte{0x3})
	address4 := common.MustBytesToAddress([]byte{0x4})

	valueDeclaration1 := ValueDeclaration{
		Name: "foo",
		Type: &sema.FunctionType{
			ReturnTypeAnnotation: &sema.TypeAnnotation{
				Type: sema.VoidType,
			},
		},
		Kind:           common.DeclarationKindFunction,
		IsConstant:     true,
		ArgumentLabels: nil,
		Available: func(location common.Location) bool {
			addressLocation, ok := location.(common.AddressLocation)
			return ok && addressLocation.Address == address2
		},
		Value: nil,
	}

	valueDeclaration2 := ValueDeclaration{
		Name: "foo",
		Type: &sema.FunctionType{
			Parameters: []*sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "n",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			ReturnTypeAnnotation: &sema.TypeAnnotation{
				Type: sema.VoidType,
			},
		},
		Kind:           common.DeclarationKindFunction,
		IsConstant:     true,
		ArgumentLabels: nil,
		Available: func(location common.Location) bool {
			addressLocation, ok := location.(common.AddressLocation)
			return ok && addressLocation.Address == address4
		},
		Value: nil,
	}

	program2 := []byte(`pub contract C2 { pub fun main() { foo() } }`)
	program3 := []byte(`pub contract C3 { pub fun main() { foo() } }`)
	program4 := []byte(`pub contract C4 { pub fun main() { foo(1) } }`)

	program1 := []byte(`
	  import 0x2
	  import 0x3
	  import 0x4

	  pub fun main() {
		  foo()
	  }
	`)

	runtime := newTestInterpreterRuntime()

	runtimeInterface := &testRuntimeInterface{
		getAccountContractCode: func(address Address, name string) (bytes []byte, err error) {
			switch address {
			case address2:
				return program2, nil
			case address3:
				return program3, nil
			case address4:
				return program4, nil
			default:
				return nil, fmt.Errorf("unknown address: %s", address.ShortHexWithPrefix())
			}
		},
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source: program1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
			PredeclaredValues: []ValueDeclaration{
				valueDeclaration1,
				valueDeclaration2,
			},
		},
	)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := checker.ExpectCheckerErrors(t, err, 2)

	// The illegal use of 'foo' in 0x3 should be reported

	var importedProgramError *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramError)
	//require.Equal(t, location3, importedProgramError.ImportLocation)
	importedErrs := checker.ExpectCheckerErrors(t, importedProgramError.Err, 1)
	require.IsType(t, &sema.NotDeclaredError{}, importedErrs[0])

	// The illegal use of 'foo' in 0x1 should be reported

	require.IsType(t, &sema.NotDeclaredError{}, errs[1])
}
