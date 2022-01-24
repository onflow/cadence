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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestArrayUpdateIndexAccess(t *testing.T) {

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

			t.Parallel()

			_, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(`
				pub contract C {
					pub %s Foo {
						%s %s x: [Int]
				
						init() {
						self.x = [3]
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

func TestDictionaryUpdateIndexAccess(t *testing.T) {

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
						%s %s x: {Int: Int}
				
						init() {
						self.x = {0: 3}
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

func TestNestedArrayUpdateIndexAccess(t *testing.T) {

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

func TestNestedDictionaryUpdateIndexAccess(t *testing.T) {

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
						%s %s x: {Int: Int}
				
						init() {
							self.x = {3: 3}
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

func TestMutateContractIndexAccess(t *testing.T) {

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
					%s %s x: [Int]
				
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

func TestContractNestedStructIndexAccess(t *testing.T) {

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
					pub let x: S
					
					pub struct S {
						%s %s y: [Int]
						init() {
							self.y = [3]
						}
					}
				
					init() {
						self.x = S()
					}
				}
				
				pub fun bar() {
					Foo.x.y[0] = 1
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

func TestContractStructInitIndexAccess(t *testing.T) {

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
					pub let x: S
					
					pub struct S {
						%s %s y: [Int]
						init() {
							self.y = [3]
						}
					}
				
					init() {
						self.x = S()
						self.x.y[1] = 2
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

func TestArrayUpdateMethodCall(t *testing.T) {

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

	type MethodCall = struct {
		Mutating bool
		Code     string
		Name     string
	}

	memberExpressions := []MethodCall{
		{Mutating: true, Code: ".append(3)", Name: "append"},
		{Mutating: false, Code: ".length", Name: "length"},
		{Mutating: false, Code: ".concat([3])", Name: "concat"},
		{Mutating: false, Code: ".contains(3)", Name: "contains"},
		{Mutating: true, Code: ".appendAll([3])", Name: "appendAll"},
		{Mutating: true, Code: ".insert(at: 0, 3)", Name: "insert"},
		{Mutating: true, Code: ".remove(at: 0)", Name: "remove"},
		{Mutating: true, Code: ".removeFirst()", Name: "removeFirst"},
		{Mutating: true, Code: ".removeLast()", Name: "removeLast"},
	}

	runTest := func(access string, declaration string, valueKind string, member MethodCall) {
		testName := fmt.Sprintf("%s %s %s %s", access, valueKind, declaration, member.Name)

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
						%s %s x: [Int]
				
						init() {
						self.x = [3]
						}
					}

					pub fun bar() {
						let foo %s Foo()
						foo.x%s
						%s
					}
				}
			`, valueKind, access, declaration, assignmentOp, member.Code, destroyStatement),
				ParseAndCheckOptions{},
			)

			if member.Mutating {
				errs := ExpectCheckerErrors(t, err, 1)
				var externalMutationError *sema.ExternalMutationError
				require.ErrorAs(t, errs[0], &externalMutationError)
			} else {
				require.NoError(t, err)
			}
		})
	}

	for _, access := range accessModifiers {
		for _, kind := range declarationKinds {
			for _, value := range valueKinds {
				for _, member := range memberExpressions {
					runTest(access, kind, value, member)
				}
			}
		}
	}
}

func TestDictionaryUpdateMethodCall(t *testing.T) {

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

	type MethodCall = struct {
		Mutating bool
		Code     string
		Name     string
	}

	memberExpressions := []MethodCall{
		{Mutating: true, Code: ".insert(key:3, 3)", Name: "insert"},
		{Mutating: false, Code: ".length", Name: "length"},
		{Mutating: false, Code: ".keys", Name: "keys"},
		{Mutating: false, Code: ".values", Name: "values"},
		{Mutating: false, Code: ".containsKey(3)", Name: "containsKey"},
		{Mutating: true, Code: ".remove(key: 0)", Name: "remove"},
	}

	runTest := func(access string, declaration string, valueKind string, member MethodCall) {
		testName := fmt.Sprintf("%s %s %s %s", access, valueKind, declaration, member.Name)

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
						%s %s x: {Int: Int}
				
						init() {
						self.x = {3: 3}
						}
					}

					pub fun bar() {
						let foo %s Foo()
						foo.x%s
						%s
					}
				}
			`, valueKind, access, declaration, assignmentOp, member.Code, destroyStatement),
				ParseAndCheckOptions{},
			)

			if member.Mutating {
				errs := ExpectCheckerErrors(t, err, 1)
				var externalMutationError *sema.ExternalMutationError
				require.ErrorAs(t, errs[0], &externalMutationError)
			} else {
				require.NoError(t, err)
			}
		})
	}

	for _, access := range accessModifiers {
		for _, kind := range declarationKinds {
			for _, value := range valueKinds {
				for _, member := range memberExpressions {
					runTest(access, kind, value, member)
				}
			}
		}
	}
}

func TestPubSetAccessModifier(t *testing.T) {
	t.Run("pub set dict", func(t *testing.T) {
		_, err := ParseAndCheckWithOptions(t,
			`
			pub contract C {
				pub struct Foo {
					pub(set) var x: {Int: Int}
			
					init() {
						self.x = {3: 3}
					}
				}

				pub fun bar() {
					let foo = Foo()
					foo.x[0] = 3
				}
			}
		`,
			ParseAndCheckOptions{},
		)
		require.NoError(t, err)

	})
}

func TestPubSetNestedAccessModifier(t *testing.T) {
	t.Run("pub set nested", func(t *testing.T) {
		_, err := ParseAndCheckWithOptions(t,
			`
			pub contract C {
				pub struct Bar {
					pub let foo: Foo
					init() { 
					   self.foo = Foo()
					}
				}
				
				pub struct Foo {
					pub(set) var x: [Int]
				
					init() {
					   self.x = [3]
					}
				}
				
				pub fun bar() {
					let bar = Bar()
					bar.foo.x[0] = 3
				}
			}
		`,
			ParseAndCheckOptions{},
		)
		require.NoError(t, err)

	})
}

func TestSelfContainingStruct(t *testing.T) {
	t.Run("pub let", func(t *testing.T) {
		_, err := ParseAndCheckWithOptions(t,
			`
			pub contract C {
				pub struct Foo {
					pub let x: {Int: Int}
			
					init() {
						self.x = {3: 3}
					}

					pub fun bar() {
						let foo = Foo()
						foo.x[0] = 3
					}
				}
			}
		`,
			ParseAndCheckOptions{},
		)
		require.NoError(t, err)

	})
}

func TestMutationThroughReference(t *testing.T) {
	t.Run("pub let", func(t *testing.T) {
		_, err := ParseAndCheckWithOptions(t,
			`
			pub fun main() {
				let foo = Foo()
				foo.ref.arr.append("y")
			  }
			  
			  pub struct Foo {
				pub let ref: &Bar
				init() {
				  self.ref = &Bar() as &Bar
				}
			  }
			  
			  pub struct Bar {
				pub let arr: [String]
				init() {
				  self.arr = ["x"]
				}
			  }
		`,
			ParseAndCheckOptions{},
		)
		errs := ExpectCheckerErrors(t, err, 1)
		var externalMutationError *sema.ExternalMutationError
		require.ErrorAs(t, errs[0], &externalMutationError)
	})
}

func TestMutationThroughAccess(t *testing.T) {
	t.Run("pub let", func(t *testing.T) {
		_, err := ParseAndCheckWithOptions(t,
			`
			pub contract C {
				pub struct Foo {
					pub let arr: [Int]
					init() {
						self.arr = [3]
					}
				}
				
				priv let foo : Foo
			
				init() {
					self.foo = Foo()
				}
			
				pub fun getFoo(): Foo {
					return self.foo
				}
			}
			
			pub fun main() {
				let a = C.getFoo()
				a.arr.append(0) // a.arr is now [3, 0]
			}
		`,
			ParseAndCheckOptions{},
		)
		errs := ExpectCheckerErrors(t, err, 1)
		var externalMutationError *sema.ExternalMutationError
		require.ErrorAs(t, errs[0], &externalMutationError)
	})
}
