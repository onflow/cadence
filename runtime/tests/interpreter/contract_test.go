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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretContractUseBeforeInitializationComplete(t *testing.T) {

	t.Parallel()

	t.Run("constructor of nested type, always qualified ", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
			`
              contract C {

                  struct S1 {

                      init() {
                          C.S2()
                      }
                  }

                  struct S2 {}

                  init() {
                      C.S1()
                  }
              }
	        `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)
	})

	t.Run("constructor of nested type, first qualified ", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
			`
              contract C {

                  struct S1 {

                      init() {
                          S2()
                      }
                  }

                  struct S2 {}

                  init() {
                      C.S1()
                  }
              }
	        `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)
	})

	t.Run("constructor of nested type, second qualified ", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
			`
              contract C {

                  struct S1 {

                      init() {
                          C.S2()
                      }
                  }

                  struct S2 {}

                  init() {
                      S1()
                  }
              }
	        `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)
	})

	t.Run("field in nested initializer", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
			`
              contract C {

                  struct S {

                      init() {
                          // use before initialization
                          C.x
                      }
                  }

                  let x: Int   

                  init() {
                      S()
                      self.x = 1
                  }
              }
	        `,
			ParseCheckAndInterpretOptions{
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.ErrorAs(t, err, &interpreter.MissingMemberValueError{})
	})
}
