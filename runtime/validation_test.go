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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

// TestRuntimeArgumentImportMissingType tests if errors produced while validating
// transaction and script arguments are gracefully handled.
// This is for example the case when an argument specifies a non-existing type,
// which results in a type loading error.
func TestRuntimeArgumentImportMissingType(t *testing.T) {

	t.Parallel()

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          transaction(value: AnyStruct) {}
        `)

		runtimeInterface := &TestRuntimeInterface{
			OnGetSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
			OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
				return nil, nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

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
				Location:  common.TransactionLocation{},
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})

	t.Run("script", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all) fun main(value: AnyStruct) {}
        `)

		runtimeInterface := &TestRuntimeInterface{
			OnGetSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
			OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
				return nil, nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

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
				Location:  common.ScriptLocation{},
			},
		)
		RequireError(t, err)

		var typeLoadingErr interpreter.TypeLoadingError
		require.ErrorAs(t, err, &typeLoadingErr)
	})
}
