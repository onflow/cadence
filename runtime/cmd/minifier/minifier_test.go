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
package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cadenceTestScript = `// This transaction is a template for a transaction that
// could be used by anyone to send tokens to another account
// that has been set up to receive tokens.
//
// The withdraw amount and the account from getAccount
// would be the parameters to the transaction

import FungibleToken from 0xFUNGIBLETOKENADDRESS
import ExampleToken from 0xTOKENADDRESS

transaction(amount: UFix64, to: Address) {

    // The Vault resource that holds the tokens that are being transferred
    let sentVault: @FungibleToken.Vault

    prepare(signer: AuthAccount) {

        // Get a reference to the signer's stored vault
        let vaultRef = signer.borrow<&ExampleToken.Vault>(from: /storage/exampleTokenVault)
			?? panic("Could not borrow reference to the owner's Vault!")

        // Withdraw tokens from the signer's stored vault
        self.sentVault <- vaultRef.withdraw(amount: amount)
    }

    execute {

        // Get the recipient's public account object
        let recipient = getAccount(to)

        // Get a reference to the recipient's Receiver
        let receiverRef = recipient.getCapability(/public/exampleTokenReceiver).borrow<&{FungibleToken.Receiver}>()
			?? panic("Could not borrow receiver reference to the recipient's Vault")

        // Deposit the withdrawn tokens in the recipient's receiver
        receiverRef.deposit(from: <-self.sentVault)
    }
}

`

const expectedOutput = `import FungibleToken from 0xFUNGIBLETOKENADDRESS
import ExampleToken from 0xTOKENADDRESS
transaction(amount: UFix64, to: Address) {
let sentVault: @FungibleToken.Vault
prepare(signer: AuthAccount) {
let vaultRef = signer.borrow<&ExampleToken.Vault>(from: /storage/exampleTokenVault)
?? panic("Could not borrow reference to the owner's Vault!")
self.sentVault <- vaultRef.withdraw(amount: amount)
}
execute {
let recipient = getAccount(to)
let receiverRef = recipient.getCapability(/public/exampleTokenReceiver).borrow<&{FungibleToken.Receiver}>()
?? panic("Could not borrow receiver reference to the recipient's Vault")
receiverRef.deposit(from: <-self.sentVault)
}
}
`

// Test to test the minifier function
func TestMinify(t *testing.T) {
	// create an input file with the test cadence script
	inputFile, err := ioutil.TempFile("", "test_*.cdc")
	require.NoError(t, err)
	inputFileName := inputFile.Name()
	_, err = inputFile.WriteString(cadenceTestScript)
	require.NoError(t, err)
	err = inputFile.Close()
	require.NoError(t, err)

	// get a valid output file path
	outputFile, err := ioutil.TempFile("", "minified_test_*.cdc")
	require.NoError(t, err)
	err = outputFile.Close()
	require.NoError(t, err)
	outputFileName := outputFile.Name()
	err = os.Remove(outputFileName)
	require.NoError(t, err)

	// call minify
	err = minify(inputFileName, outputFileName)

	// assert no error
	require.NoError(t, err)

	defer os.Remove(outputFileName)

	// read the output file contents and assert the contents
	actualOutput, err := ioutil.ReadFile(outputFileName)
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(actualOutput))
}
