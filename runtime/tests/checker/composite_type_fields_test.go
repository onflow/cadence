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
	"fmt"
	"testing"

	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"
)

func TestCompositeTypeFields(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		code       string
		errorTypes []error
	}{

		"int is a storable field": {`
		contract Controller {

			var n: Int

			init(){
				self.n = 1
			}
		}
			`,
			nil,
		},

		"function is not a storable field": {`
		contract Controller {

			var fn: (():Int)

			init(){
				self.fn = fun(): Int {
					return 1
				};
			}

			pub fun test(): Int {
					return 2
			}
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
			},
		},

		"[Int] is a storable field": {`
		contract Controller {

			var xs: [Int]

			init(){
				self.xs = [1, 2, 3]
			}
		}
			`,
			nil,
		},

		"{Int: String} is a storable field": {`
		contract Controller {

			var m: {Int: String}

			init(){
				self.m = {}
			}
		}
			`,
			nil,
		},

		// owner is a predeclare member with PublicAccount? type,
		// can't assign it to a non-storable field
		"PublicAccount is a not storable field": {`
		resource R {
			var p: PublicAccount?

			init(){
				self.p = self.owner
			}
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
			},
		},

		"{Int: function} is not a storable field": {`
		contract Controller {

			var m: {Int: ((): Int)}

			init(){
				self.m = {
					1: fun(): Int {
						return 1
					}
				}
			}
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
			},
		},

		"[function] is not a storable field": {`
		contract Controller {

			var operators: [(():Int)]

			init(){
				self.operators = []
			}
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
			},
		},

		"function field for struct is not storable": {`
		struct MyStruct {
			pub var fn: (():Int)

			init() {
				self.fn = fun(): Int {
					return 1
				};
			};
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
			},
		},

		"path field is not storable": {`
		struct MyStruct {
			pub var fn: (():Int)

			init() {
				self.fn = fun(): Int {
					return 1
				}
			}
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
			},
		},

		"nested field for resource is not storable": {`
		contract S {
			let r : @R

			resource R {
				// function field in nested composite type is not allowed
				pub var fn: (():Int)

				init() {
					self.fn = fun(): Int {
						return 1
					}
				}
			}
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
				&sema.MissingInitializerError{},
			},
		},

		"resource interface is storable if all fields are storable": {`
		resource interface RI {
			var r: Int
			var s: String
		}
			`,
			nil,
		},

		"resource interface is not storable if one field is not storable": {`
		resource interface RI {
			var r: Int
			var p: PublicAccount // PublicAccount is not a storable field
		}

		resource R {
			var m : @{String: {RI}}
		}
			`,
			[]error{
				&sema.FieldTypeNotStorableError{},
				&sema.MissingInitializerError{},
				&sema.MissingDestructorError{},
			},
		},
	}

	for caseName, testcase := range cases {
		t.Run(caseName, func(t *testing.T) {
			_, err := ParseAndCheck(t, testcase.code)

			errmsg := fmt.Sprintf("failed test case: %v\n", testcase.code)

			if testcase.errorTypes == nil {
				if err != nil {
					cmd.PrettyPrintError(err, "", map[string]string{"": testcase.code})
				}
				// print the failed the cadence code if test case was broken
				require.NoError(t, err, errmsg)
			} else {
				errs := ExpectCheckerErrors(t, err, len(testcase.errorTypes))
				for i, errType := range testcase.errorTypes {
					require.IsType(t, errType, errs[i])
				}
			}
		})
	}

	t.Run("check error message", func(t *testing.T) {
		testcase := cases["function is not a storable field"]
		_, err := ParseAndCheck(t, testcase.code)
		require.Error(t, err)

		checkerError, _ := err.(*sema.CheckerError)
		require.Equal(t, "field fn is not storable, type: ((): Int)",
			checkerError.ChildErrors()[0].Error())
	})
}
