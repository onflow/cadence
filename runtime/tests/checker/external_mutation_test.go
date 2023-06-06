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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckArrayUpdateIndexAccess(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	valueKinds := []common.CompositeKind{
		common.CompositeKindStructure,
		common.CompositeKindResource,
	}

	runTest := func(access ast.Access, declaration common.DeclarationKind, valueKind common.CompositeKind) {
		testName := fmt.Sprintf("%s %s %s", access.Keyword(), valueKind.Keyword(), declaration.Keywords())

		assignmentOp := "="
		var destroyStatement string
		if valueKind == common.CompositeKindResource {
			assignmentOp = "<- create"
			destroyStatement = "destroy foo"
		}

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract C {
                    access(all) %s Foo {
                        %s %s x: [Int]
                
                        init() {
                        self.x = [3]
                        }
                    }

                    access(all) fun bar() {
                        let foo %s Foo()
                        foo.x[0] = 3
                        %s
                    }
                }
            `, valueKind.Keyword(), access.Keyword(), declaration.Keywords(), assignmentOp, destroyStatement),
			)

			errs := RequireCheckerErrors(t, err, 1)
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

func TestCheckDictionaryUpdateIndexAccess(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	valueKinds := []common.CompositeKind{
		common.CompositeKindStructure,
		common.CompositeKindResource,
	}

	runTest := func(access ast.Access, declaration common.DeclarationKind, valueKind common.CompositeKind) {
		testName := fmt.Sprintf("%s %s %s", access.Keyword(), valueKind.Keyword(), declaration.Keywords())

		assignmentOp := "="
		var destroyStatement string
		if valueKind == common.CompositeKindResource {
			assignmentOp = "<- create"
			destroyStatement = "destroy foo"
		}

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract C {
                    access(all) %s Foo {
                        %s %s x: {Int: Int}
                
                        init() {
                        self.x = {0: 3}
                        }
                    }

                    access(all) fun bar() {
                        let foo %s Foo()
                        foo.x[0] = 3
                        %s
                    }
                }
            `, valueKind.Keyword(), access.Keyword(), declaration.Keywords(), assignmentOp, destroyStatement),
			)

			errs := RequireCheckerErrors(t, err, 1)
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

func TestCheckNestedArrayUpdateIndexAccess(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	runTest := func(access ast.Access, declaration common.DeclarationKind) {
		testName := fmt.Sprintf("%s %s", access.Keyword(), declaration.Keywords())

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract C {
                    access(all) struct Bar {
                        access(all) let foo: Foo
                        init() {
                            self.foo = Foo()
                        }
                    }

                    access(all) struct Foo {
                        %s %s x: [Int]
                
                        init() {
                            self.x = [3]
                        }
                    }

                    access(all) fun bar() {
                        let bar = Bar()
                        bar.foo.x[0] = 3
                    }
                }
            `, access.Keyword(), declaration.Keywords()),
			)

			errs := RequireCheckerErrors(t, err, 1)
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

func TestCheckNestedDictionaryUpdateIndexAccess(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	runTest := func(access ast.Access, declaration common.DeclarationKind) {
		testName := fmt.Sprintf("%s %s", access.Keyword(), declaration.Keywords())

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract C {
                    access(all) struct Bar {
                        access(all) let foo: Foo
                        init() {
                            self.foo = Foo()
                        }
                    }

                    access(all) struct Foo {
                        %s %s x: {Int: Int}
                
                        init() {
                            self.x = {3: 3}
                        }
                    }

                    access(all) fun bar() {
                        let bar = Bar()
                        bar.foo.x[0] = 3
                    }
                }
            `, access.Keyword(), declaration.Keywords()),
			)

			errs := RequireCheckerErrors(t, err, 1)
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

func TestCheckMutateContractIndexAccess(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	runTest := func(access ast.Access, declaration common.DeclarationKind) {
		testName := fmt.Sprintf("%s %s", access.Keyword(), declaration.Keywords())

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract Foo {
                    %s %s x: [Int]
                
                    init() {
                        self.x = [3]
                    }
                }
                
                access(all) fun bar() {
                    Foo.x[0] = 1
                }
            `, access.Keyword(), declaration.Keywords()),
			)

			expectedErrors := 1
			if access == ast.AccessContract {
				expectedErrors++
			}

			errs := RequireCheckerErrors(t, err, expectedErrors)
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

func TestCheckContractNestedStructIndexAccess(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	runTest := func(access ast.Access, declaration common.DeclarationKind) {
		testName := fmt.Sprintf("%s %s", access.Keyword(), declaration.Keywords())

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract Foo {
                    access(all) let x: S
                    
                    access(all) struct S {
                        %s %s y: [Int]
                        init() {
                            self.y = [3]
                        }
                    }
                
                    init() {
                        self.x = S()
                    }
                }
                
                access(all) fun bar() {
                    Foo.x.y[0] = 1
                }                
            `, access.Keyword(), declaration.Keywords()),
			)

			expectedErrors := 1
			if access == ast.AccessContract {
				expectedErrors++
			}

			errs := RequireCheckerErrors(t, err, expectedErrors)
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

func TestCheckContractStructInitIndexAccess(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	runTest := func(access ast.Access, declaration common.DeclarationKind) {
		testName := fmt.Sprintf("%s %s", access.Keyword(), declaration.Keywords())

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract Foo {
                    access(all) let x: S
                    
                    access(all) struct S {
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
            `, access.Keyword(), declaration.Keywords()),
			)

			errs := RequireCheckerErrors(t, err, 1)
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

func TestCheckArrayUpdateMethodCall(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	valueKinds := []common.CompositeKind{
		common.CompositeKindStructure,
		common.CompositeKindResource,
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

	runTest := func(access ast.Access, declaration common.DeclarationKind, valueKind common.CompositeKind, member MethodCall) {
		testName := fmt.Sprintf("%s %s %s %s", access.Keyword(), valueKind.Keyword(), declaration.Keywords(), member.Name)

		assignmentOp := "="
		var destroyStatement string
		if valueKind == common.CompositeKindResource {
			assignmentOp = "<- create"
			destroyStatement = "destroy foo"
		}

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract C {
                    access(all) %s Foo {
                        %s %s x: [Int]
                
                        init() {
                        self.x = [3]
                        }
                    }

                    access(all) fun bar() {
                        let foo %s Foo()
                        foo.x%s
                        %s
                    }
                }
            `, valueKind.Keyword(), access.Keyword(), declaration.Keywords(), assignmentOp, member.Code, destroyStatement),
			)

			if member.Mutating {
				errs := RequireCheckerErrors(t, err, 1)
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

func TestCheckDictionaryUpdateMethodCall(t *testing.T) {

	t.Parallel()

	accessModifiers := []ast.Access{
		ast.AccessAll,
		ast.AccessAccount,
		ast.AccessContract,
	}

	declarationKinds := []common.DeclarationKind{
		common.DeclarationKindConstant,
		common.DeclarationKindVariable,
	}

	valueKinds := []common.CompositeKind{
		common.CompositeKindStructure,
		common.CompositeKindResource,
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

	runTest := func(access ast.Access, declaration common.DeclarationKind, valueKind common.CompositeKind, member MethodCall) {
		testName := fmt.Sprintf("%s %s %s %s", access.Keyword(), valueKind.Keyword(), declaration.Keywords(), member.Name)

		assignmentOp := "="
		var destroyStatement string
		if valueKind == common.CompositeKindResource {
			assignmentOp = "<- create"
			destroyStatement = "destroy foo"
		}

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                access(all) contract C {
                    access(all) %s Foo {
                        %s %s x: {Int: Int}
                
                        init() {
                            self.x = {3: 3}
                        }
                    }

                    access(all) fun bar() {
                        let foo %s Foo()
                        foo.x%s
                        %s
                    }
                }
            `, valueKind.Keyword(), access.Keyword(), declaration.Keywords(), assignmentOp, member.Code, destroyStatement),
			)

			if member.Mutating {
				errs := RequireCheckerErrors(t, err, 1)
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

func TestCheckSelfContainingStruct(t *testing.T) {

	t.Parallel()

	t.Run("access(all) let", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) contract C {
                access(all) struct Foo {
                    access(all) let x: {Int: Int}
            
                    init() {
                        self.x = {3: 3}
                    }

                    access(all) fun bar() {
                        let foo = Foo()
                        foo.x[0] = 3
                    }
                }
            }
        `,
		)
		require.NoError(t, err)

	})
}

func TestCheckMutationThroughReference(t *testing.T) {

	t.Parallel()

	t.Run("access(all) let", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun main() {
                let foo = Foo()
                foo.ref.arr.append("y")
              }
              
              access(all) struct Foo {
                access(all) let ref: &Bar
                init() {
                  self.ref = &Bar() as &Bar
                }
              }
              
              access(all) struct Bar {
                access(all) let arr: [String]
                init() {
                  self.arr = ["x"]
                }
              }
        `,
		)
		errs := RequireCheckerErrors(t, err, 1)
		var externalMutationError *sema.ExternalMutationError
		require.ErrorAs(t, errs[0], &externalMutationError)
	})
}

func TestCheckMutationThroughInnerReference(t *testing.T) {

	t.Parallel()

	t.Run("access(all) let", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun main() {
                let foo = Foo()
                var arrayRef = &foo.ref.arr as &[String]
                arrayRef[0] = "y"
              }
              
              access(all) struct Foo {
                access(all) let ref: &Bar
                init() {
                  self.ref = &Bar() as &Bar
                }
              }
              
              access(all) struct Bar {
                access(all) let arr: [String]
                init() {
                  self.arr = ["x"]
                }
              }
        `,
		)
		require.NoError(t, err)
	})
}

func TestCheckMutationThroughAccess(t *testing.T) {

	t.Parallel()

	t.Run("access(all) let", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) contract C {
                access(all) struct Foo {
                    access(all) let arr: [Int]
                    init() {
                        self.arr = [3]
                    }
                }
                
                access(self) let foo : Foo
            
                init() {
                    self.foo = Foo()
                }
            
                access(all) fun getFoo(): Foo {
                    return self.foo
                }
            }
            
            access(all) fun main() {
                let a = C.getFoo()
                a.arr.append(0) // a.arr is now [3, 0]
            }
        `,
		)
		errs := RequireCheckerErrors(t, err, 1)
		var externalMutationError *sema.ExternalMutationError
		require.ErrorAs(t, errs[0], &externalMutationError)
	})
}
