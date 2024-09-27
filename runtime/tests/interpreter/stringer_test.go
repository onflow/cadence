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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestStringerBasic(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
		access(all)
		struct Example: Stringer {
			view fun toString():String {
				return "example"
			}  
		}
		fun test() :String {
			return Example().toString()
		}
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("example"),
		result,
	)
}

func TestStringerBuiltIn(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
		access(all)
		fun test() :String {
			let v = 1
			return v.toString()
		}
	`)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("1"),
		result,
	)
}
