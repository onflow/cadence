/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func TestContractUpdateValidation(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime(
		WithContractUpdateValidationEnabled(true),
	)

	newDeployTransaction := func(function, name, code string) []byte {
		return []byte(fmt.Sprintf(`
			transaction {
				prepare(signer: AuthAccount) {
					signer.contracts.%s(name: "%s", code: "%s".decodeHex())
				}
			}`,
			function,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	deployAndUpdate := func(t *testing.T, name string, oldCode string, newCode string) error {

		accountCode := map[common.LocationID][]byte{

			"A.73dd87ae00edff1e.MojoAssetdefinition": []byte(`
import MojoProject from 0x73dd87ae00edff1e

pub contract MojoAssetdefinition {}

`),

			"A.73dd87ae00edff1e.MojoProject": []byte(`
import MojoCommunityVault from 0x73dd87ae00edff1e
import FlowToken from 0x7e60df042a9c0868

access(all) contract MojoProject {}
`),

			"A.73dd87ae00edff1e.MojoCommunityVault": []byte(`

import MojoToken from 0x73dd87ae00edff1e
import MojoAsset from 0x73dd87ae00edff1e

pub contract MojoCommunityVault {}

`),
			"A.73dd87ae00edff1e.MojoToken": []byte(`
import FungibleToken from 0x9a0766d93b6608b7
import MojoAdminInterfaces from 0x73dd87ae00edff1e

pub contract MojoToken {}
`),

			"A.73dd87ae00edff1e.MojoAdminInterfaces": []byte(`
pub contract MojoAdminInterfaces {}

`),

			"A.73dd87ae00edff1e.MojoAsset": []byte(`
import MojoAssetdefinition from 0x73dd87ae00edff1e
import MojoToken from 0x73dd87ae00edff1e

pub contract MojoAsset {}

`),
		}
		var events []cadence.Event
		runtimeInterface := getMockedRuntimeInterfaceForTxUpdate(t, accountCode, events)
		nextTransactionLocation := newTransactionLocationGenerator()

		deployTx1 := newDeployTransaction(sema.AuthAccountContractsTypeAddFunctionName, name, oldCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		deployTx2 := newDeployTransaction(sema.AuthAccountContractsTypeUpdateExperimentalFunctionName, name, newCode)
		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTx2,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		return err
	}

	t.Run("Mojo", func(t *testing.T) {
		const oldCode = `
import MojoAssetdefinition from 0x73dd87ae00edff1e
import  MojoToken from 0x73dd87ae00edff1e

pub contract MojoAsset {}
`

		const newCode = `

import MojoAssetdefinition from 0x73dd87ae00edff1e
import  MojoToken from 0x73dd87ae00edff1e
import MojoCommunityVault from 0x73dd87ae00edff1e

pub contract MojoAsset {}
}
`

		err := deployAndUpdate(t, "MojoAsset", oldCode, newCode)
		require.Error(t, err)

		//cause := getErrorCause(t, err, "Test1")
		//assertFieldTypeMismatchError(t, cause, "Test1", "a", "String", "Int")
	})
}

func assertDeclTypeChangeError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	oldKind common.DeclarationKind,
	newKind common.DeclarationKind,
) {

	require.Error(t, err)
	require.IsType(t, &InvalidDeclarationKindChangeError{}, err)
	declTypeChangeError := err.(*InvalidDeclarationKindChangeError)
	assert.Equal(
		t,
		fmt.Sprintf("trying to convert %s `%s` to a %s", oldKind.Name(), erroneousDeclName, newKind.Name()),
		declTypeChangeError.Error(),
	)
}

func assertExtraneousFieldError(t *testing.T, err error, erroneousDeclName string, fieldName string) {
	require.Error(t, err)
	require.IsType(t, &ExtraneousFieldError{}, err)
	extraFieldError := err.(*ExtraneousFieldError)
	assert.Equal(t, fmt.Sprintf("found new field `%s` in `%s`", fieldName, erroneousDeclName), extraFieldError.Error())
}

func assertFieldTypeMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	fieldName string,
	expectedType string,
	foundType string,
) {

	require.Error(t, err)
	require.IsType(t, &FieldMismatchError{}, err)
	fieldMismatchError := err.(*FieldMismatchError)
	assert.Equal(
		t,
		fmt.Sprintf("mismatching field `%s` in `%s`", fieldName, erroneousDeclName),
		fieldMismatchError.Error(),
	)

	assert.IsType(t, &TypeMismatchError{}, fieldMismatchError.err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		fieldMismatchError.err.Error(),
	)
}

func assertConformanceMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	expectedType string,
	foundType string,
) {

	require.Error(t, err)
	require.IsType(t, &ConformanceMismatchError{}, err)
	conformanceMismatchError := err.(*ConformanceMismatchError)
	assert.Equal(
		t,
		fmt.Sprintf("conformances does not match in `%s`", erroneousDeclName),
		conformanceMismatchError.Error(),
	)

	assert.IsType(t, &TypeMismatchError{}, conformanceMismatchError.err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		conformanceMismatchError.err.Error(),
	)
}

func getErrorCause(t *testing.T, err error, contractName string) error {
	updateErr := getContractUpdateError(t, err)
	assert.Equal(t, fmt.Sprintf("cannot update contract `%s`", contractName), updateErr.Error())

	require.Equal(t, 1, len(updateErr.ChildErrors()))
	childError := updateErr.ChildErrors()[0]

	return childError
}

func getContractUpdateError(t *testing.T, err error) *ContractUpdateError {
	require.Error(t, err)
	require.IsType(t, Error{}, err)
	runtimeError := err.(Error)

	require.IsType(t, interpreter.Error{}, runtimeError.Err)
	interpreterError := runtimeError.Err.(interpreter.Error)

	require.IsType(t, &InvalidContractDeploymentError{}, interpreterError.Err)
	deploymentError := interpreterError.Err.(*InvalidContractDeploymentError)

	require.IsType(t, &ContractUpdateError{}, deploymentError.Err)
	return deploymentError.Err.(*ContractUpdateError)
}

func getMockedRuntimeInterfaceForTxUpdate(
	t *testing.T,
	accountCodes map[common.LocationID][]byte,
	events []cadence.Event,
) *testRuntimeInterface {

	return &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.BytesToAddress([]byte{0x42})}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) (err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}
}

func TestContractUpdateValidationDisabled(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime(
		WithContractUpdateValidationEnabled(false),
	)

	newDeployTransaction := func(function, name, code string) []byte {
		return []byte(fmt.Sprintf(`
			transaction {
				prepare(signer: AuthAccount) {
					signer.contracts.%s(name: "%s", code: "%s".decodeHex())
				}
			}`,
			function,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	accountCode := map[common.LocationID][]byte{}
	var events []cadence.Event
	runtimeInterface := getMockedRuntimeInterfaceForTxUpdate(t, accountCode, events)
	nextTransactionLocation := newTransactionLocationGenerator()

	deployAndUpdate := func(t *testing.T, name string, oldCode string, newCode string) error {
		deployTx1 := newDeployTransaction(sema.AuthAccountContractsTypeAddFunctionName, name, oldCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		deployTx2 := newDeployTransaction(sema.AuthAccountContractsTypeUpdateExperimentalFunctionName, name, newCode)
		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTx2,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		return err
	}

	t.Run("change field type", func(t *testing.T) {
		const oldCode = `
			pub contract Test1 {
				pub var a: String
				init() {
					self.a = "hello"
				}
      		}`

		const newCode = `
			pub contract Test1 {
				pub var a: Int
				init() {
					self.a = 0
				}
			}`

		err := deployAndUpdate(t, "Test1", oldCode, newCode)
		require.NoError(t, err)
	})
}
