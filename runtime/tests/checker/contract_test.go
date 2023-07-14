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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckInvalidContractAccountField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract Test {
          let account: AuthAccount

          init(account: AuthAccount) {
              self.account = account
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckInvalidContractInterfaceAccountField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract interface Test {
          let account: AuthAccount
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckInvalidContractAccountFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract Test {
          fun account() {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckInvalidContractInterfaceAccountFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract interface Test {
          fun account()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}

func TestCheckContractAccountFieldUse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract Test {

          init() {
              self.account.address
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckContractInterfaceAccountFieldUse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract interface Test {

          fun test() {
              pre { self.account.address == Address(0x42) }
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidContractAccountFieldInitialization(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract Test {

          init(account: AuthAccount) {
              self.account = account
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[0])
}

func TestCheckInvalidContractAccountFieldAccess(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract Test {}

      let test = Test.account
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidAccessError{}, errs[0])
}

func TestCheckContractAccountFieldUseInitialized(t *testing.T) {

	t.Parallel()

	code := `
      contract Test {
          let address: Address

          init() {
              // field 'account' can be used, as it is considered initialized
              self.address = self.account.address
          }

          fun test(): Address {
              return self.account.address
          }
      }

      let address1 = Test.address
      let address2 = Test.test()
    `
	_, err := ParseAndCheck(t, code)

	require.NoError(t, err)
}

func TestCheckInvalidContractMoveToFunction(t *testing.T) {

	t.Parallel()

	for _, name := range []string{"self", "C"} {

		t.Run(name, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      contract C {

                          fun test() {
                              use(%s)
                          }
                      }

                      fun use(_ c: C) {}
                    `,
					name,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})
	}
}

func TestCheckInvalidContractMoveInVariableDeclaration(t *testing.T) {

	t.Parallel()

	for _, name := range []string{"self", "C"} {

		t.Run(name, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      contract C {

                          fun test() {
                              let x = %s
                          }
                      }
                    `,
					name,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})
	}
}

func TestCheckInvalidContractMoveReturnFromFunction(t *testing.T) {

	t.Parallel()

	for _, name := range []string{"self", "C"} {

		t.Run(name, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      contract C {

                          fun test(): C {
                              return %s
                          }
                      }
                    `,
					name,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})
	}
}

func TestCheckInvalidContractMoveIntoArrayLiteral(t *testing.T) {

	t.Parallel()

	for _, name := range []string{"self", "C"} {

		t.Run(name, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      contract C {

                          fun test() {
                              let txs = [%s]
                          }
                      }
                    `,
					name,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})
	}
}

func TestCheckInvalidContractMoveIntoDictionaryLiteral(t *testing.T) {

	t.Parallel()

	for _, name := range []string{"self", "C"} {

		t.Run(name, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      contract C {

                          fun test() {
                              let txs = {"C": %s}
                          }
                      }
                    `,
					name,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidMoveError{}, errs[0])
		})
	}
}

func TestCheckContractNestedDeclarationOrderOutsideInside(t *testing.T) {

	t.Parallel()

	for _, isInterface := range []bool{true, false} {

		interfaceKeyword := ""
		if isInterface {
			interfaceKeyword = "interface"
		}

		body := ""
		if !isInterface {
			body = "{}"
		}

		extraFunction := ""
		if !isInterface {
			extraFunction = `
		      fun callGoNew() {
                  let r <- create R()
                  r.go()
                  destroy r
              }
            `
		}

		annotationType := "R"
		if isInterface {
			annotationType = "{R}"
		}

		t.Run(interfaceKeyword, func(t *testing.T) {

			code := fmt.Sprintf(
				`
                  contract C {

                      fun callGoExisting(r: @%[1]s) {
                          r.go()
                          destroy r
                      }

                      %[2]s

                      resource %[3]s R {
                          fun go() %[4]s
                      }
                  }
                `,
				annotationType,
				extraFunction,
				interfaceKeyword,
				body,
			)
			_, err := ParseAndCheck(t, code)

			require.NoError(t, err)
		})
	}
}

func TestCheckContractNestedDeclarationOrderInsideOutside(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract C {

          fun go() {}

          resource R {
              fun callGo() {
                  C.go()
              }
          }
      }
    `)

	require.NoError(t, err)
}

// TestCheckContractNestedDeclarationsComplex tests
// - Using inner types in functions outside (both type in parameter and constructor)
// - Using outer functions in inner types' functions
// - Mutually using sibling types
func TestCheckContractNestedDeclarationsComplex(t *testing.T) {

	t.Parallel()

	interfacePossibilities := []bool{true, false}

	compositeKinds := []common.CompositeKind{
		common.CompositeKindStructure,
		common.CompositeKindResource,
	}

	for _, contractIsInterface := range interfacePossibilities {
		for _, firstKind := range compositeKinds {
			for _, firstIsInterface := range interfacePossibilities {
				for _, secondKind := range compositeKinds {
					for _, secondIsInterface := range interfacePossibilities {

						contractInterfaceKeyword := ""
						if contractIsInterface {
							contractInterfaceKeyword = "interface"
						}

						firstInterfaceKeyword := ""
						if firstIsInterface {
							firstInterfaceKeyword = "interface"
						}

						secondInterfaceKeyword := ""
						if secondIsInterface {
							secondInterfaceKeyword = "interface"
						}

						testName := fmt.Sprintf(
							"contract_%s/%s_%s/%s_%s",
							contractInterfaceKeyword,
							firstKind.Keyword(),
							firstInterfaceKeyword,
							secondKind.Keyword(),
							secondInterfaceKeyword,
						)

						bodyUsingFirstOutside := ""
						if !contractIsInterface {
							if secondIsInterface {
								bodyUsingFirstOutside = fmt.Sprintf(
									"{ %s a }",
									firstKind.DestructionKeyword(),
								)
							} else {
								bodyUsingFirstOutside = fmt.Sprintf(
									"{ a.localB(%[1]s %[2]s B()); %[3]s a }",
									secondKind.MoveOperator(),
									secondKind.ConstructionKeyword(),
									firstKind.DestructionKeyword(),
								)
							}
						}

						bodyUsingSecondOutside := ""
						if !contractIsInterface {
							if firstIsInterface {
								bodyUsingSecondOutside = fmt.Sprintf(
									"{ %s b }",
									secondKind.DestructionKeyword(),
								)
							} else {
								bodyUsingSecondOutside = fmt.Sprintf(
									"{ b.localA(%[1]s %[2]s A()); %[3]s b }",
									firstKind.MoveOperator(),
									firstKind.ConstructionKeyword(),
									secondKind.DestructionKeyword(),
								)
							}
						}

						bodyUsingFirstInsideFirst := ""
						bodyUsingSecondInsideFirst := ""
						bodyUsingFirstInsideSecond := ""
						bodyUsingSecondInsideSecond := ""

						if !contractIsInterface && !firstIsInterface {
							bodyUsingFirstInsideFirst = fmt.Sprintf(
								"{ C.localBeforeA(%s a) }",
								firstKind.MoveOperator(),
							)
							bodyUsingSecondInsideFirst = fmt.Sprintf(
								"{ C.localBeforeB(%s b)  }",
								secondKind.MoveOperator(),
							)
						}

						if !contractIsInterface && !secondIsInterface {
							bodyUsingFirstInsideSecond = fmt.Sprintf(
								"{ C.qualifiedAfterA(%s a) }",
								firstKind.MoveOperator(),
							)
							bodyUsingSecondInsideSecond = fmt.Sprintf(
								"{ C.qualifiedAfterB(%s b)  }",
								secondKind.MoveOperator(),
							)
						}

						firstQualifiedTypeAnnotation := "C.A"
						firstLocalTypeAnnotation := "A"
						if firstIsInterface {
							switch firstKind {
							case common.CompositeKindResource:
								firstQualifiedTypeAnnotation = fmt.Sprintf(
									"{%s}",
									firstQualifiedTypeAnnotation,
								)
								firstLocalTypeAnnotation = fmt.Sprintf(
									"{%s}",
									firstLocalTypeAnnotation,
								)

							case common.CompositeKindStructure:
								firstQualifiedTypeAnnotation = fmt.Sprintf(
									"{%s}",
									firstQualifiedTypeAnnotation,
								)
								firstLocalTypeAnnotation = fmt.Sprintf(
									"{%s}",
									firstLocalTypeAnnotation,
								)

							}
						}

						secondQualifiedTypeAnnotation := "C.B"
						secondLocalTypeAnnotation := "B"
						if secondIsInterface {
							switch secondKind {
							case common.CompositeKindResource:
								secondQualifiedTypeAnnotation = fmt.Sprintf(
									"{%s}",
									secondQualifiedTypeAnnotation,
								)
								secondLocalTypeAnnotation = fmt.Sprintf(
									"{%s}",
									secondLocalTypeAnnotation,
								)

							case common.CompositeKindStructure:
								secondQualifiedTypeAnnotation = fmt.Sprintf(
									"{%s}",
									secondQualifiedTypeAnnotation,
								)
								secondLocalTypeAnnotation = fmt.Sprintf(
									"{%s}",
									secondLocalTypeAnnotation,
								)
							}
						}

						t.Run(testName, func(t *testing.T) {

							code := fmt.Sprintf(
								`
                                  contract %[1]s C {

                                      fun qualifiedBeforeA(_ a: %[4]s%[14]s) %[12]s
                                      fun localBeforeA(_ a: %[4]s%[16]s) %[12]s

                                      fun qualifiedBeforeB(_ b: %[7]s%[15]s) %[13]s
                                      fun localBeforeB(_ b: %[7]s%[17]s) %[13]s

                                      %[2]s %[3]s A {
                                          fun qualifiedB(_ b: %[7]s%[15]s) %[9]s
                                          fun localB(_ b: %[7]s%[17]s) %[9]s

                                          fun qualifiedA(_ a: %[4]s%[14]s) %[8]s
                                          fun localA(_ a: %[4]s%[16]s) %[8]s
                                      }

                                      %[5]s %[6]s B {
                                          fun qualifiedA(_ a: %[4]s%[14]s) %[10]s
                                          fun localA(_ a: %[4]s%[16]s) %[10]s

                                          fun qualifiedB(_ b: %[7]s%[15]s) %[11]s
                                          fun localB(_ b: %[7]s%[17]s) %[11]s
                                      }

                                      fun qualifiedAfterA(_ a: %[4]s%[14]s) %[12]s
                                      fun localAfterA(_ a: %[4]s%[16]s) %[12]s

                                      fun qualifiedAfterB(_ b: %[7]s%[15]s) %[13]s
                                      fun localAfterB(_ b: %[7]s%[17]s) %[13]s
                                  }
                                `,
								contractInterfaceKeyword,
								firstKind.Keyword(),
								firstInterfaceKeyword,
								firstKind.Annotation(),
								secondKind.Keyword(),
								secondInterfaceKeyword,
								secondKind.Annotation(),
								bodyUsingFirstInsideFirst,
								bodyUsingSecondInsideFirst,
								bodyUsingFirstInsideSecond,
								bodyUsingSecondInsideSecond,
								bodyUsingFirstOutside,
								bodyUsingSecondOutside,
								firstQualifiedTypeAnnotation,
								secondQualifiedTypeAnnotation,
								firstLocalTypeAnnotation,
								secondLocalTypeAnnotation,
							)
							_, err := ParseAndCheck(t, code)

							require.NoError(t, err)
						})
					}
				}
			}
		}
	}
}

func TestCheckInvalidContractNestedTypeShadowing(t *testing.T) {

	t.Parallel()

	type test struct {
		name        string
		code        string
		isInterface bool
	}

	tests := []test{
		{name: "event", code: `event Test()`, isInterface: false},
	}

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {

		// Contracts can not be nested
		if kind == common.CompositeKindContract {
			continue
		}

		for _, isInterface := range []bool{true, false} {
			keywords := kind.Keyword()

			if isInterface && kind == common.CompositeKindAttachment {
				continue
			}

			if isInterface {
				keywords += " interface"
			}

			var baseType string
			if kind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			code := fmt.Sprintf(`%s Test %s {}`, keywords, baseType)

			tests = append(tests, test{
				name:        keywords,
				code:        code,
				isInterface: isInterface,
			})
		}
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(`
                      contract Test {
                          %s
                      }
                    `,
					test.code,
				),
			)

			// If the nested element is an interface, there will only be an error
			// for the redeclared type.
			//
			// If the nested element is a concrete type, there will also be an error
			// for the redeclared value (constructor).

			expectedErrors := 1
			if !test.isInterface {
				expectedErrors += 1
			}

			errs := RequireCheckerErrors(t, err, expectedErrors)

			for i := 0; i < expectedErrors; i++ {
				assert.IsType(t, &sema.RedeclarationError{}, errs[i])
			}
		})
	}
}

func TestCheckBadContractNesting(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "contract signatureAlgorithm { resource interface payer { contract foo : payer { contract foo { contract foo { } contract foo { contract interface account { } } contract account { } } } } }")

	errs := RequireCheckerErrors(t, err, 14)

	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[1])
	assert.IsType(t, &sema.RedeclarationError{}, errs[2])
	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[3])
	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[4])
	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[5])
	assert.IsType(t, &sema.RedeclarationError{}, errs[6])
	assert.IsType(t, &sema.RedeclarationError{}, errs[7])
	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[8])
	assert.IsType(t, &sema.RedeclarationError{}, errs[9])
	assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[10])
	assert.IsType(t, &sema.MissingConformanceError{}, errs[11])
	assert.IsType(t, &sema.RedeclarationError{}, errs[12])
	assert.IsType(t, &sema.RedeclarationError{}, errs[13])
}

func TestCheckContractEnumAccessRestricted(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheckWithOptions(t, "contract foo{}let x = foo!",
		ParseAndCheckOptions{
			Config: &sema.Config{
				AccessCheckMode: sema.AccessCheckModeStrict,
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.MissingAccessModifierError{}, errs[0])
	assert.IsType(t, &sema.MissingAccessModifierError{}, errs[1])
}
