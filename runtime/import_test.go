/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
)

func TestRuntimeCyclicImport(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

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
		programChecked: func(location common.Location, duration time.Duration) {
			checkCount += 1
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	location := nextTransactionLocation()
	context := Context{
		Interface: runtimeInterface,
		Location:  location,
	}
	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		context,
	)
	require.Error(t, err)

	// Script

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	require.Len(t, checkerErr.ChildErrors(), 1)
	childErr := checkerErr.ChildErrors()[0]

	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t, childErr, &importedProgramErr)

	// P1

	var checkerErr2 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr.Err, &checkerErr2)

	require.Len(t, checkerErr2.ChildErrors(), 1)
	childErr2 := checkerErr2.ChildErrors()[0]

	var importedProgramErr2 *sema.ImportedProgramError
	require.ErrorAs(t, childErr2, &importedProgramErr2)

	// P2

	var checkerErr3 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr2.Err, &checkerErr3)

	require.Len(t, checkerErr3.ChildErrors(), 1)
	childErr3 := checkerErr3.ChildErrors()[0]

	var importedProgramErr3 *sema.ImportedProgramError
	require.ErrorAs(t, childErr3, &importedProgramErr3)

	require.IsType(t, importedProgramErr3.Err, &sema.CyclicImportsError{})
}
