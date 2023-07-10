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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretCharacterUtf8Field(t *testing.T) {

	t.Parallel()

	runTest := func(t *testing.T, code string, expectedValues ...interpreter.Value) {
		inter := parseCheckAndInterpret(t, fmt.Sprintf(`
		fun test(): [UInt8] {
			let c: Character = "%s"
			return c.utf8
		}
	  `, code))

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
				common.ZeroAddress,
				expectedValues...,
			),
			result,
		)
	}

	runTest(t, `a`, interpreter.NewUnmeteredUInt8Value(97))
	runTest(t, `F`, interpreter.NewUnmeteredUInt8Value(70))
	runTest(t, `\u{1F490}`,
		interpreter.NewUnmeteredUInt8Value(240),
		interpreter.NewUnmeteredUInt8Value(159),
		interpreter.NewUnmeteredUInt8Value(146),
		interpreter.NewUnmeteredUInt8Value(144),
	)
	runTest(t, "👪",
		interpreter.NewUnmeteredUInt8Value(240),
		interpreter.NewUnmeteredUInt8Value(159),
		interpreter.NewUnmeteredUInt8Value(145),
		interpreter.NewUnmeteredUInt8Value(170),
	)
}
