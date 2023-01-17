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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

// TestRuntimeArgumentImportMissingType tests if errors produced while validating
// transaction and script arguments are gracefully handled.
// This is for example the case when an argument specifies a non-existing type,
// which results in a type loading error.
func TestRuntimeArgumentImportMissingType(t *testing.T) {

	t.Parallel()

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
          transaction(value: AnyStruct) {}
        `)

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				return nil, nil
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		nextTransactionLocation := newTransactionLocationGenerator()
		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
				Arguments: encodeArgs([]cadence.Value{
					cadence.Struct{}.
						WithType(&cadence.StructType{
							Location: common.AddressLocation{
								Address: common.ZeroAddress,
								Name:    "Foo",
							},
							QualifiedIdentifier: "Foo.Bar",
						}),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})

	t.Run("script", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
          pub fun main(value: AnyStruct) {}
        `)

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				return nil, nil
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		nextTransactionLocation := newTransactionLocationGenerator()
		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
				Arguments: encodeArgs([]cadence.Value{
					cadence.Struct{}.
						WithType(&cadence.StructType{
							Location: common.AddressLocation{
								Address: common.ZeroAddress,
								Name:    "Foo",
							},
							QualifiedIdentifier: "Foo.Bar",
						}),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})
}
