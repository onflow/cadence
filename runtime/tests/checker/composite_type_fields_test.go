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

package checker

import (
	"testing"

	"github.com/onflow/cadence/runtime/cmd"
	"github.com/stretchr/testify/require"
)

func TestCompositeTypeFields(t *testing.T) {
	cases := map[string]struct {
		code   string
		result bool
	}{

		"int is a storable field": {`
		contract Controller {

			var n: Int

			init(){
				self.n = 1;
			}
		}
			`,
			true,
		},

		"function is not a storable field": {`
		contract Controller {

			var fn: (():Int)

			init(){
				self.fn = fun(): Int {
					return 1;
				};
			}
		}
			`,
			false,
		},

		"[int] is a storable field": {`
		contract Controller {

			var xs: [Int]

			init(){
				self.xs = [1, 2, 3];
			}
		}
			`,
			true,
		},

		"[function] is not a storable field": {`
		contract Controller {

			var operators: [(():Int)]

			init(){
				self.operators = [];
			}
		}
			`,
			false,
		},

		"storable field is only for contract": {`
		struct MyStruct {
			pub var fn: (():Int)

			init() {
				self.fn = fun(): Int {
					return 1;
				};
			};
		}
			`,
			true,
		},
	}

	for caseName, testcase := range cases {
		t.Run(caseName, func(t *testing.T) {
			_, err := ParseAndCheck(t, testcase.code)

			if testcase.result {
				if err != nil {
					cmd.PrettyPrintError(err, "", map[string]string{"": testcase.code})
				}
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
