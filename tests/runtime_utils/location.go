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

package runtime_utils

import (
	"encoding/binary"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
)

func NewLocationGenerator[T ~[32]byte]() func() T {
	var count uint64
	return func() T {
		t := T{}
		newCount := atomic.AddUint64(&count, 1)
		binary.LittleEndian.PutUint64(t[:], newCount)
		return t
	}
}

func NewTransactionLocationGenerator() func() common.TransactionLocation {
	return NewLocationGenerator[common.TransactionLocation]()
}

func NewScriptLocationGenerator() func() common.ScriptLocation {
	return NewLocationGenerator[common.ScriptLocation]()
}

func NewSingleIdentifierLocationResolver(t testing.TB) func(
	identifiers []runtime.Identifier,
	location runtime.Location,
) (
	[]runtime.ResolvedLocation,
	error,
) {
	return func(
		identifiers []runtime.Identifier,
		location runtime.Location,
	) (
		[]sema.ResolvedLocation,
		error,
	) {
		require.Len(t, identifiers, 1)
		require.IsType(t, common.AddressLocation{}, location)

		return []sema.ResolvedLocation{
			{
				Location: common.AddressLocation{
					Address: location.(common.AddressLocation).Address,
					Name:    identifiers[0].Identifier,
				},
				Identifiers: identifiers,
			},
		}, nil
	}
}

func MultipleIdentifierLocationResolver(
	identifiers []ast.Identifier,
	location common.Location,
) (
	result []sema.ResolvedLocation,
	err error,
) {

	// Resolve each identifier as an address location

	for _, identifier := range identifiers {
		result = append(result, sema.ResolvedLocation{
			Location: common.AddressLocation{
				Address: location.(common.AddressLocation).Address,
				Name:    identifier.Identifier,
			},
			Identifiers: []ast.Identifier{
				identifier,
			},
		})
	}

	return
}
