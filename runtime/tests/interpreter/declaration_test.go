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

package interpreter_test

import (
	"testing"
)

func TestInterpretForwardReferenceCall(t *testing.T) {

	t.Parallel()

	t.Run("variable before composite", func(t *testing.T) {

		t.Parallel()

		_ = parseCheckAndInterpret(t,
			`
              let s = S()

              struct S {}
	    `)
	})

	t.Run("variable before function", func(t *testing.T) {

		t.Parallel()

		_ = parseCheckAndInterpret(t,
			`
              let g = f()

              fun f() {}
	    `)
	})

	t.Run("indirect forward reference", func(t *testing.T) {

		t.Parallel()

		// Here, x has a forward reference to y,
		// through f and g

		_ = parseCheckAndInterpret(t,
			`
              fun f(): Int {
                  return g()
              }

              let x = f()
              let y = 0

              fun g(): Int {
                  return y
              }
	    `)
	})

}
