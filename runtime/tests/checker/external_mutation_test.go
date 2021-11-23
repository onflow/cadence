/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"
)

func TestArrayUpdate(t *testing.T) {

	t.Parallel()

	accessModifiers := []string{
		"pub",
		"access(account)",
		"access(contract)",
	}

	declarationKinds := []string{
		"let",
		"var",
	}

	valueKinds := []string{
		"struct",
		"resource",
	}

	runTest := func(access string, declaration string, valueKind string) {
		testName := fmt.Sprintf("%s %s %s", access, valueKind, declaration)

		assignmentOp := "="
		var destroyStatement string
		if valueKind == "resource" {
			assignmentOp = "<- create"
			destroyStatement = "destroy foo"
		}

		t.Run(testName, func(t *testing.T) {
			_, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(`
				pub contract C {
					pub %s Foo {
						%s %s x : [Int]
				
						init() {
						self.x = [3];
						}
					}

					pub fun bar() {
						let foo %s Foo()
						foo.x[0] = 3
						%s
					}
				}
			`, valueKind, access, declaration, assignmentOp, destroyStatement),
				ParseAndCheckOptions{},
			)

			errs := ExpectCheckerErrors(t, err, 1)
			var externalMutationError *sema.ExternalMutationError
			require.ErrorAs(t, errs[0], &externalMutationError)
		})
	}

	for _, access := range accessModifiers {
		for _, kind := range declarationKinds {
			for _, value := range valueKinds {
				runTest(access, kind, value)
			}
		}
	}

}

func TestDictionaryUpdate(t *testing.T) {

	t.Parallel()

	accessModifiers := []string{
		"pub",
		"access(account)",
		"access(contract)",
	}

	declarationKinds := []string{
		"let",
		"var",
	}

	valueKinds := []string{
		"struct",
		"resource",
	}

	runTest := func(access string, declaration string, valueKind string) {
		testName := fmt.Sprintf("%s %s %s", access, valueKind, declaration)

		assignmentOp := "="
		var destroyStatement string
		if valueKind == "resource" {
			assignmentOp = "<- create"
			destroyStatement = "destroy foo"
		}

		t.Run(testName, func(t *testing.T) {
			_, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(`
				pub contract C {
					pub %s Foo {
						%s %s x : {Int: Int}
				
						init() {
						self.x = {0:3};
						}
					}

					pub fun bar() {
						let foo %s Foo()
						foo.x[0] = 3
						%s
					}
				}
			`, valueKind, access, declaration, assignmentOp, destroyStatement),
				ParseAndCheckOptions{},
			)

			errs := ExpectCheckerErrors(t, err, 1)
			var externalMutationError *sema.ExternalMutationError
			require.ErrorAs(t, errs[0], &externalMutationError)
		})
	}

	for _, access := range accessModifiers {
		for _, kind := range declarationKinds {
			for _, value := range valueKinds {
				runTest(access, kind, value)
			}
		}
	}

}

func TestNestedArrayUpdate(t *testing.T) {

	t.Parallel()

	accessModifiers := []string{
		"pub",
		"access(account)",
		"access(contract)",
	}

	declarationKinds := []string{
		"let",
		"var",
	}

	runTest := func(access string, declaration string) {
		testName := fmt.Sprintf("%s struct %s", access, declaration)

		t.Run(testName, func(t *testing.T) {
			_, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(`
				pub contract C {
					pub struct Bar {
						pub let foo: Foo
						init() {
							self.foo = Foo()
						}
					}

					pub struct Foo {
						%s %s x : [Int]
				
						init() {
							self.x = [3]
						}
					}

					pub fun bar() {
						let bar = Bar()
						bar.foo.x[0] = 3
					}
				}
			`, access, declaration),
				ParseAndCheckOptions{},
			)

			errs := ExpectCheckerErrors(t, err, 1)
			var externalMutationError *sema.ExternalMutationError
			require.ErrorAs(t, errs[0], &externalMutationError)
		})
	}

	for _, access := range accessModifiers {
		for _, kind := range declarationKinds {
			runTest(access, kind)
		}
	}
}

func TestNestedDictionaryUpdate(t *testing.T) {

	t.Parallel()

	accessModifiers := []string{
		"pub",
		"access(account)",
		"access(contract)",
	}

	declarationKinds := []string{
		"let",
		"var",
	}

	runTest := func(access string, declaration string) {
		testName := fmt.Sprintf("%s struct %s", access, declaration)

		t.Run(testName, func(t *testing.T) {
			_, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(`
				pub contract C {
					pub struct Bar {
						pub let foo: Foo
						init() {
							self.foo = Foo()
						}
					}

					pub struct Foo {
						%s %s x : {Int: Int}
				
						init() {
							self.x = {3:3}
						}
					}

					pub fun bar() {
						let bar = Bar()
						bar.foo.x[0] = 3
					}
				}
			`, access, declaration),
				ParseAndCheckOptions{},
			)

			errs := ExpectCheckerErrors(t, err, 1)
			var externalMutationError *sema.ExternalMutationError
			require.ErrorAs(t, errs[0], &externalMutationError)
		})
	}

	for _, access := range accessModifiers {
		for _, kind := range declarationKinds {
			runTest(access, kind)
		}
	}
}

func TestMutateContract(t *testing.T) {

	t.Parallel()

	accessModifiers := []string{
		"pub",
		"access(account)",
		"access(contract)",
	}

	declarationKinds := []string{
		"let",
		"var",
	}

	runTest := func(access string, declaration string) {
		testName := fmt.Sprintf("%s struct %s", access, declaration)

		t.Run(testName, func(t *testing.T) {
			_, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(`
				pub contract Foo {
					%s %s x : [Int]
				
					init() {
						self.x = [3]
					}
				}
				
				pub fun bar() {
					Foo.x[0] = 1
				}
			`, access, declaration),
				ParseAndCheckOptions{},
			)

			expectedErrors := 1
			if access == "access(contract)" {
				expectedErrors++
			}

			errs := ExpectCheckerErrors(t, err, expectedErrors)
			if expectedErrors > 1 {
				var accessError *sema.InvalidAccessError
				require.ErrorAs(t, errs[expectedErrors-2], &accessError)
			}
			var externalMutationError *sema.ExternalMutationError
			require.ErrorAs(t, errs[expectedErrors-1], &externalMutationError)
		})
	}

	for _, access := range accessModifiers {
		for _, kind := range declarationKinds {
			runTest(access, kind)
		}
	}
}
