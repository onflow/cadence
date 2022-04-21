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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckToString(t *testing.T) {

	t.Parallel()

	for _, numberOrAddressType := range append(
		sema.AllNumberTypes[:],
		&sema.AddressType{},
	) {

		ty := numberOrAddressType

		t.Run(ty.String(), func(t *testing.T) {

			t.Parallel()

			checker, err := parseAndCheckWithTestValue(t,
				`
                  let res = test.toString()
                `,
				ty,
			)

			require.NoError(t, err)

			resType := RequireGlobalValue(t, checker.Elaboration, "res")

			assert.Equal(t,
				sema.StringType,
				resType,
			)
		})
	}
}

func TestCheckToBytes(t *testing.T) {

	t.Parallel()

	t.Run("Address", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let address: Address = 0x1
          let res = address.toBytes()
        `)

		require.NoError(t, err)

		resType := RequireGlobalValue(t, checker.Elaboration, "res")

		assert.Equal(t,
			sema.ByteArrayType,
			resType,
		)
	})
}

func TestCheckToBigEndianBytes(t *testing.T) {

	for _, ty := range sema.AllNumberTypes {

		t.Run(ty.String(), func(t *testing.T) {

			checker, err := parseAndCheckWithTestValue(t,
				`
                  let res = test.toBigEndianBytes()
                `,
				ty,
			)

			require.NoError(t, err)

			resType := RequireGlobalValue(t, checker.Elaboration, "res")

			assert.Equal(t,
				sema.ByteArrayType,
				resType,
			)
		})
	}
}
