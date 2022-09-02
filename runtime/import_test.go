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
	"time"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
)

func TestRuntimeCyclicImport(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	imported1 := []byte(`
      import p2
    `)

	imported2 := []byte(`
      import p1
    `)

	script := []byte(`
      import p1

      pub fun main() {}
    `)

	var checkCount int

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.IdentifierLocation("p1"):
				return imported1, nil
			case common.IdentifierLocation("p2"):
				return imported2, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		programChecked: func(location Location, duration time.Duration) {
			checkCount += 1
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	location := nextTransactionLocation()
	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  location,
		},
	)
	require.Error(t, err)

	require.Contains(t, err.Error(), "cyclic import of `p1`")

	// Script

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := checker.ExpectCheckerErrors(t, checkerErr, 1)

	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr)

	// P1

	var checkerErr2 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr.Err, &checkerErr2)

	errs = checker.ExpectCheckerErrors(t, checkerErr2, 1)

	var importedProgramErr2 *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr2)

	// P2

	var checkerErr3 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr2.Err, &checkerErr3)

	errs = checker.ExpectCheckerErrors(t, checkerErr3, 1)

	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}

func TestRuntimeExport(t *testing.T) {

	t.Parallel()

	t.Run("interpreted function (invalid)", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
            pub fun f() {}

            pub fun main(): AnyStruct {
                return f
            }
        `)

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: &testRuntimeInterface{},
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)
		require.IsType(t, Error{}, err)
	})
}
