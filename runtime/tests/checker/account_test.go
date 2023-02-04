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
	"github.com/onflow/cadence/runtime/stdlib"
)

func ParseAndCheckAccount(t *testing.T, code string) (*sema.Checker, error) {

	constantDeclaration := func(name string, ty sema.Type) stdlib.StandardLibraryValue {
		return stdlib.StandardLibraryValue{
			Name: name,
			Type: ty,
			Kind: common.DeclarationKindConstant,
		}
	}
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(constantDeclaration("authAccount", sema.AuthAccountType))
	baseValueActivation.DeclareValue(constantDeclaration("publicAccount", sema.PublicAccountType))

	return ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivation: baseValueActivation,
			},
		},
	)
}

func TestCheckAccount_save(t *testing.T) {

	t.Parallel()

	testImplicitTypeArgument := func(domain common.PathDomain) {

		domainName := domain.Identifier()
		domainIdentifier := domain.Identifier()

		testName := func(kind string) string {
			return fmt.Sprintf(
				"implicit type argument, %s, %s",
				domainName,
				kind,
			)
		}

		t.Run(testName("resource"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      resource R {}

                      fun test() {
                          let r <- create R()
                          authAccount.save(<-r, to: /%s/r)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {
				require.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			}
		})

		t.Run(testName("struct"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      struct S {}

                      fun test() {
                          let s = S()
                          authAccount.save(s, to: /%s/s)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {
				require.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			}
		})
	}

	testExplicitTypeArgumentCorrect := func(domain common.PathDomain) {

		domainName := domain.Identifier()
		domainIdentifier := domain.Identifier()

		testName := func(kind string) string {
			return fmt.Sprintf(
				"explicit type argument, %s, %s",
				domainName,
				kind,
			)
		}

		t.Run(testName("resource"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      resource R {}

                      fun test() {
                          let r <- create R()
                          authAccount.save<@R>(<-r, to: /%s/r)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {
				require.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			}
		})

		t.Run(testName("struct"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      struct S {}

                      fun test() {
                          let s = S()
                          authAccount.save<S>(s, to: /%s/s)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {
				require.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			}
		})
	}

	testExplicitTypeArgumentIncorrect := func(domain common.PathDomain) {

		domainName := domain.Identifier()
		domainIdentifier := domain.Identifier()

		testName := func(kind string) string {
			return fmt.Sprintf(
				"explicit type argument, incorrect, %s, %s",
				domainName,
				kind,
			)
		}

		t.Run(testName("resource"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      resource R {}

                      resource T {}

                      fun test() {
                          let r <- create R()
                          authAccount.save<@T>(<-r, to: /%s/r)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {

				errs := RequireCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
			} else {
				errs := RequireCheckerErrors(t, err, 3)

				require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
				require.IsType(t, &sema.TypeMismatchError{}, errs[2])
			}
		})

		t.Run(testName("struct"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      struct S {}

                      struct T {}

                      fun test() {
                          let s = S()
                          authAccount.save<T>(s, to: /%s/s)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {

				errs := RequireCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
			} else {
				errs := RequireCheckerErrors(t, err, 3)

				require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
				require.IsType(t, &sema.TypeMismatchError{}, errs[2])
			}
		})
	}

	testInvalidNonStorable := func(domain common.PathDomain) {

		domainName := domain.Identifier()
		domainIdentifier := domain.Identifier()

		testName := func(kind string) string {
			return fmt.Sprintf(
				"invalid non-storable, %s, %s",
				domainName,
				kind,
			)
		}

		t.Run(testName("explicit type argument"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      fun one(): Int {
                          return 1
                      }

                      fun test() {
                          authAccount.save<fun(): Int>(one, to: /%s/one)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			} else {
				errs := RequireCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
			}
		})

		t.Run(testName("implicit type argument"), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      fun one(): Int {
                          return 1
                      }

                      fun test() {
                          authAccount.save(one, to: /%s/one)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			} else {
				errs := RequireCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
			}
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		testImplicitTypeArgument(domain)
		testExplicitTypeArgumentCorrect(domain)
		testExplicitTypeArgumentIncorrect(domain)
		testInvalidNonStorable(domain)
	}
}

func TestCheckAccount_typeAt(t *testing.T) {

	t.Parallel()

	test := func(domain common.PathDomain) {
		t.Run(fmt.Sprintf("type %s", domain.Identifier()), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
						let t: Type? = authAccount.type(at: /%s/r)
					`,
					domain.Identifier(),
				),
			)

			if domain == common.PathDomainStorage {

				require.NoError(t, err)

				typ := RequireGlobalValue(t, checker.Elaboration, "t")

				require.Equal(t,
					&sema.OptionalType{
						Type: sema.MetaType,
					},
					typ,
				)

			} else {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			}
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		test(domain)
	}
}

func TestCheckAccount_load(t *testing.T) {

	t.Parallel()

	testMissingTypeArguments := func(domain common.PathDomain) {

		testName := fmt.Sprintf(
			"missing type argument, %s",
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      let s = authAccount.load(from: /%s/s)
                    `,
					domain.Identifier(),
				),
			)

			if domain == common.PathDomainStorage {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

			} else {
				errs := RequireCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
			}
		})
	}

	testExplicitTypeArgument := func(domain common.PathDomain) {

		testName := fmt.Sprintf(
			"explicit type argument, %s",
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("resource", func(t *testing.T) {

				t.Parallel()

				checker, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let r <- authAccount.load<@R>(from: /%s/r)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {

					require.NoError(t, err)

					rType := RequireGlobalType(t, checker.Elaboration, "R")
					rValueType := RequireGlobalValue(t, checker.Elaboration, "r")

					require.Equal(t,
						&sema.OptionalType{
							Type: rType,
						},
						rValueType,
					)

				} else {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})

			t.Run("struct", func(t *testing.T) {

				t.Parallel()

				checker, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let s = authAccount.load<S>(from: /%s/s)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {

					require.NoError(t, err)

					sType := RequireGlobalType(t, checker.Elaboration, "S")
					sValueType := RequireGlobalValue(t, checker.Elaboration, "s")

					require.Equal(t,
						&sema.OptionalType{
							Type: sType,
						},
						sValueType,
					)
				} else {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		testMissingTypeArguments(domain)
		testExplicitTypeArgument(domain)
	}
}

func TestCheckAccount_copy(t *testing.T) {

	t.Parallel()

	testMissingTypeArgument := func(domain common.PathDomain) {

		testName := fmt.Sprintf(
			"missing type argument, %s",
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      struct S {}

                      let s = authAccount.copy(from: /%s/s)
                    `,
					domain.Identifier(),
				),
			)

			if domain == common.PathDomainStorage {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

			} else {
				errs := RequireCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
			}
		})
	}

	testExplicitTypeArgument := func(domain common.PathDomain) {

		testName := fmt.Sprintf(
			"explicit type argument, %s",
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("struct", func(t *testing.T) {

				t.Parallel()

				checker, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let s = authAccount.copy<S>(from: /%s/s)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {
					require.NoError(t, err)

					sType := RequireGlobalType(t, checker.Elaboration, "S")
					sValueType := RequireGlobalValue(t, checker.Elaboration, "s")

					require.Equal(t,
						&sema.OptionalType{
							Type: sType,
						},
						sValueType,
					)

				} else {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})

			t.Run("resource", func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let r <- authAccount.copy<@R>(from: /%s/r)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

				} else {
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
					require.IsType(t, &sema.TypeMismatchError{}, errs[1])
				}
			})
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		testMissingTypeArgument(domain)
		testExplicitTypeArgument(domain)
	}
}

func TestCheckAccount_borrow(t *testing.T) {

	t.Parallel()

	testMissingTypeArgument := func(domain common.PathDomain) {

		testName := fmt.Sprintf(
			"missing type argument, %s",
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("resource", func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          let r = authAccount.borrow(from: /%s/r)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

				} else {
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
					require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
				}
			})

			t.Run("struct", func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          let s = authAccount.borrow(from: /%s/s)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

				} else {
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
					require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
				}
			})
		})
	}

	testExplicitTypeArgumentReference := func(domain common.PathDomain, auth bool) {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf(
			"explicit type argument, %s reference, %s",
			authKeyword,
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("resource", func(t *testing.T) {

				t.Parallel()

				checker, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let r = authAccount.borrow<%s &R>(from: /%s/r)
                        `,
						authKeyword,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {

					require.NoError(t, err)

					rType := RequireGlobalType(t, checker.Elaboration, "R")
					rValueType := RequireGlobalValue(t, checker.Elaboration, "r")

					require.Equal(t,
						&sema.OptionalType{
							Type: &sema.ReferenceType{
								Authorized: auth,
								Type:       rType,
							},
						},
						rValueType,
					)
				} else {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})

			t.Run("struct", func(t *testing.T) {

				t.Parallel()

				checker, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let s = authAccount.borrow<%s &S>(from: /%s/s)
                        `,
						authKeyword,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {
					require.NoError(t, err)

					sType := RequireGlobalType(t, checker.Elaboration, "S")
					sValueType := RequireGlobalValue(t, checker.Elaboration, "s")

					require.Equal(t,
						&sema.OptionalType{
							Type: &sema.ReferenceType{
								Authorized: auth,
								Type:       sType,
							},
						},
						sValueType,
					)
				} else {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})
		})
	}

	testExplicitTypeArgumentNonReference := func(domain common.PathDomain) {

		testName := fmt.Sprintf(
			"explicit type argument, non-reference type, %s",
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("resource", func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let r <- authAccount.borrow<@R>(from: /%s/r)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {

					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
					require.IsType(t, &sema.TypeMismatchError{}, errs[1])
				}
			})

			t.Run("struct", func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let s = authAccount.borrow<S>(from: /%s/s)
                        `,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {

					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
					require.IsType(t, &sema.TypeMismatchError{}, errs[1])
				}
			})
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		testMissingTypeArgument(domain)

		for _, auth := range []bool{false, true} {
			testExplicitTypeArgumentReference(domain, auth)
		}

		testExplicitTypeArgumentNonReference(domain)
	}
}

func TestCheckAccount_link(t *testing.T) {

	t.Parallel()

	testMissingTypeArgument := func(domain common.PathDomain) {

		testName := fmt.Sprintf(
			"missing type argument, %s",
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      fun test(): Capability? {
                          return authAccount.link(/%s/r, target: /storage/r)
                      }
                    `,
					domain.Identifier(),
				),
			)

			switch domain {
			case common.PathDomainPrivate, common.PathDomainPublic:
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

			default:
				errs := RequireCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
			}
		})
	}

	testExplicitTypeArgumentReference := func(domain common.PathDomain, auth bool) {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf(
			"explicit type argument, %s reference, %s",
			authKeyword,
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("resource", func(t *testing.T) {

				t.Parallel()

				typeArguments := fmt.Sprintf("<%s &R>", authKeyword)

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          resource R {}

                          fun test(): Capability%[1]s? {
                              return authAccount.link%[1]s(/%[2]s/r, target: /storage/r)
                          }
                        `,
						typeArguments,
						domain.Identifier(),
					),
				)

				switch domain {
				case common.PathDomainPrivate, common.PathDomainPublic:
					require.NoError(t, err)

				default:
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})

			t.Run("struct", func(t *testing.T) {

				t.Parallel()

				typeArguments := fmt.Sprintf("<%s &S>", authKeyword)

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          struct S {}

                          fun test(): Capability%[1]s? {
                              return authAccount.link%[1]s(/%[2]s/s, target: /storage/s)
                          }
                        `,
						typeArguments,
						domain.Identifier(),
					),
				)

				switch domain {
				case common.PathDomainPrivate, common.PathDomainPublic:
					require.NoError(t, err)

				default:
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})
		})
	}

	testExplicitTypeArgumentTarget := func(domain, targetDomain common.PathDomain) {

		testName := fmt.Sprintf(
			"explicit type argument, non-reference type, %s -> %s",
			domain.Identifier(),
			targetDomain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("resource", func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          resource R {}

                          fun test(): Capability? {
                              return authAccount.link<@R>(/%s/r, target: /%s/r)
                          }
                        `,
						domain.Identifier(),
						targetDomain.Identifier(),
					),
				)

				switch domain {
				case common.PathDomainPrivate, common.PathDomainPublic:
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

				default:
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
					require.IsType(t, &sema.TypeMismatchError{}, errs[1])
				}
			})

			t.Run("struct", func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(
						`
                          struct S {}

                          fun test(): Capability? {
                              return authAccount.link<S>(/%s/s, target: /%s/s)
                          }
                        `,
						domain.Identifier(),
						targetDomain.Identifier(),
					),
				)

				switch domain {
				case common.PathDomainPrivate, common.PathDomainPublic:
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

				default:
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
					require.IsType(t, &sema.TypeMismatchError{}, errs[1])
				}
			})
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		testMissingTypeArgument(domain)

		for _, auth := range []bool{false, true} {
			testExplicitTypeArgumentReference(domain, auth)
		}

		for _, targetDomain := range common.AllPathDomainsByIdentifier {
			testExplicitTypeArgumentTarget(domain, targetDomain)
		}
	}
}

func TestCheckAccount_linkAccount(t *testing.T) {

	t.Parallel()

	test := func(domain common.PathDomain, enabled bool) {

		testName := fmt.Sprintf("%s, %v", domain.Identifier(), enabled)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			code := fmt.Sprintf(`
                  resource R {}

                  fun test(authAccount: AuthAccount): Capability<&AuthAccount>? {
                      return authAccount.linkAccount(/%s/r)
                  }
                `,
				domain.Identifier(),
			)

			_, err := ParseAndCheckWithOptions(t,
				code,
				ParseAndCheckOptions{
					Config: &sema.Config{
						AccountLinkingEnabled: enabled,
					},
				},
			)

			if enabled {
				switch domain {
				case common.PathDomainPrivate, common.PathDomainPublic:
					require.NoError(t, err)

				default:
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
			}
		})
	}

	for _, enabled := range []bool{true, false} {
		for _, domain := range common.AllPathDomainsByIdentifier {
			test(domain, enabled)
		}
	}
}

func TestCheckAccount_unlink(t *testing.T) {

	t.Parallel()

	test := func(domain common.PathDomain) {

		t.Run(domain.Identifier(), func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      fun test() {
                          authAccount.unlink(/%s/r)
                      }
                    `,
					domain.Identifier(),
				),
			)

			switch domain {
			case common.PathDomainPrivate, common.PathDomainPublic:
				require.NoError(t, err)

			default:
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			}
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		test(domain)
	}
}

func TestCheckAccount_getLinkTarget(t *testing.T) {

	t.Parallel()

	test := func(domain common.PathDomain, accountType, accountVariable string) {

		testName := fmt.Sprintf(
			"%s.getLinkTarget: %s",
			accountType,
			domain.Identifier(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      let path: Path? = %s.getLinkTarget(/%s/r)
                    `,
					accountVariable,
					domain.Identifier(),
				),
			)

			switch domain {
			case common.PathDomainPrivate, common.PathDomainPublic:
				require.NoError(t, err)

			default:
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			}
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {

		for accountType, accountVariable := range map[string]string{
			"AuthAccount":   "authAccount",
			"PublicAccount": "publicAccount",
		} {
			test(domain, accountType, accountVariable)
		}
	}
}

func TestCheckAccount_getCapability(t *testing.T) {

	t.Parallel()

	test := func(typed bool, accountType string, domain common.PathDomain, accountVariable string) {

		typedPrefix := ""
		if !typed {
			typedPrefix = "un"
		}

		testName := fmt.Sprintf(
			"%s.getCapability: %s, %styped",
			accountType,
			domain.Identifier(),
			typedPrefix,
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			capabilitySuffix := ""
			if typed {
				capabilitySuffix = "<&Int>"
			}

			code := fmt.Sprintf(
				`
	              fun test(): Capability%[3]s {
	                  return %[1]s.getCapability%[3]s(/%[2]s/r)
	              }

                  let cap = test()
	            `,
				accountVariable,
				domain.Identifier(),
				capabilitySuffix,
			)

			checker, err := ParseAndCheckAccount(t, code)

			switch domain {
			case common.PathDomainPrivate:

				if accountType == "PublicAccount" {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

					return
				} else {
					require.NoError(t, err)
				}

			case common.PathDomainPublic:
				require.NoError(t, err)

			default:
				errs := RequireCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])

				return
			}

			var expectedBorrowType sema.Type
			if typed {
				expectedBorrowType = &sema.ReferenceType{
					Type: sema.IntType,
				}
			}

			capType := RequireGlobalValue(t, checker.Elaboration, "cap")

			assert.Equal(t,
				&sema.CapabilityType{
					BorrowType: expectedBorrowType,
				},
				capType,
			)
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {

		for accountType, accountVariable := range map[string]string{
			"AuthAccount":   "authAccount",
			"PublicAccount": "publicAccount",
		} {

			for _, typed := range []bool{true, false} {

				test(typed, accountType, domain, accountVariable)
			}
		}
	}
}

func TestCheckAccount_BalanceFields(t *testing.T) {
	t.Parallel()

	for accountType, accountVariable := range map[string]string{
		"AuthAccount":   "authAccount",
		"PublicAccount": "publicAccount",
	} {

		for _, fieldName := range []string{
			"balance",
			"availableBalance",
		} {

			testName := fmt.Sprintf(
				"%s.%s",
				accountType,
				fieldName,
			)

			t.Run(testName, func(t *testing.T) {

				code := fmt.Sprintf(
					`
	                      fun test(): UFix64 {
	                          return %s.%s
	                      }

                          let amount = test()
	                    `,
					accountVariable,
					fieldName,
				)
				checker, err := ParseAndCheckAccount(
					t,
					code,
				)

				require.NoError(t, err)

				amountType := RequireGlobalValue(t, checker.Elaboration, "amount")

				assert.Equal(t, sema.UFix64Type, amountType)
			})
		}
	}
}

func TestCheckAccount_StorageFields(t *testing.T) {
	t.Parallel()

	for accountType, accountVariable := range map[string]string{
		"AuthAccount":   "authAccount",
		"PublicAccount": "publicAccount",
	} {

		for _, fieldName := range []string{
			"storageUsed",
			"storageCapacity",
		} {

			testName := fmt.Sprintf(
				"%s.%s",
				accountType,
				fieldName,
			)

			t.Run(testName, func(t *testing.T) {

				code := fmt.Sprintf(
					`
	                      fun test(): UInt64 {
	                          return %s.%s
	                      }

                          let amount = test()
	                    `,
					accountVariable,
					fieldName,
				)
				checker, err := ParseAndCheckAccount(
					t,
					code,
				)

				require.NoError(t, err)

				amountType := RequireGlobalValue(t, checker.Elaboration, "amount")

				assert.Equal(t, sema.UInt64Type, amountType)
			})
		}
	}
}

func TestAuthAccountContracts(t *testing.T) {

	t.Parallel()

	t.Run("contracts type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
          let contracts: AuthAccount.Contracts = authAccount.contracts
	    `)

		require.NoError(t, err)
	})

	t.Run("contracts names", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
          let names: [String] = authAccount.contracts.names
	    `)

		require.NoError(t, err)
	})

	t.Run("update contracts names", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test() {
                authAccount.contracts.names = ["foo"]
            }
	    `)

		errors := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errors[0])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errors[1])
	})

	t.Run("get contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract? {
                return authAccount.contracts.get(name: "foo")
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("borrow contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &C? {
                return authAccount.contracts.borrow<&C>(name: "foo")
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("invalid borrow contract: missing type argument", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &AnyStruct? {
                return authAccount.contracts.borrow(name: "foo")
            }
	    `)

		errors := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeParameterTypeInferenceError{}, errors[0])
	})

	t.Run("add contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract {
                return authAccount.contracts.add(name: "foo", code: "012".decodeHex())
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("update contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract {
                return authAccount.contracts.update__experimental(name: "foo", code: "012".decodeHex())
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("remove contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract? {
                return authAccount.contracts.remove(name: "foo")
            }
	    `)

		require.NoError(t, err)
	})

}

func TestPublicAccountContracts(t *testing.T) {

	t.Parallel()

	t.Run("contracts type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            let contracts: PublicAccount.Contracts = publicAccount.contracts
	    `)

		require.NoError(t, err)
	})

	t.Run("contracts names", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            let names: [String] = publicAccount.contracts.names
	    `)

		require.NoError(t, err)
	})

	t.Run("update contracts names", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test() {
                publicAccount.contracts.names = ["foo"]
            }
	    `)

		errors := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errors[0])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errors[1])
	})

	t.Run("get contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract? {
                return publicAccount.contracts.get(name: "foo")
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("borrow contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &C? {
                return publicAccount.contracts.borrow<&C>(name: "foo")
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("invalid borrow contract: missing type argument", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &AnyStruct? {
                return publicAccount.contracts.borrow(name: "foo")
            }
	    `)

		errors := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeParameterTypeInferenceError{}, errors[0])
	})

	t.Run("add contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract {
                return publicAccount.contracts.add(name: "foo", code: "012".decodeHex())
            }
	    `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredMemberError{}, errors[0])
		notDeclaredError := errors[0].(*sema.NotDeclaredMemberError)
		assert.Equal(t, "add", notDeclaredError.Name)
	})

	t.Run("update contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract {
                return publicAccount.contracts.update__experimental(name: "foo", code: "012".decodeHex())
            }
	    `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredMemberError{}, errors[0])
		notDeclaredError := errors[0].(*sema.NotDeclaredMemberError)
		assert.Equal(t, "update__experimental", notDeclaredError.Name)
	})

	t.Run("remove contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract {
                return publicAccount.contracts.remove(name: "foo")
            }
	    `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredMemberError{}, errors[0])
		notDeclaredError := errors[0].(*sema.NotDeclaredMemberError)
		assert.Equal(t, "remove", notDeclaredError.Name)
	})

}

func TestCheckAccountPaths(t *testing.T) {

	t.Parallel()
	t.Run("capitalized", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			let paths = authAccount.StoragePaths
		`,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var notDeclaredError *sema.NotDeclaredMemberError
		require.ErrorAs(t, errors[0], &notDeclaredError)

		assert.Equal(t, "StoragePaths", notDeclaredError.Name)
	})

	t.Run("annotation", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			let publicPaths: [PublicPath] = authAccount.publicPaths
			let privatePaths: [PrivatePath] = authAccount.privatePaths
			let storagePaths: [StoragePath] = authAccount.storagePaths
		`,
		)

		require.NoError(t, err)
	})

	t.Run("supertype annotation", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			let publicPaths: [Path] = authAccount.publicPaths
			let privatePaths: [CapabilityPath] = authAccount.privatePaths
			let storagePaths: [Path] = authAccount.storagePaths
		`,
		)

		require.NoError(t, err)
	})

	t.Run("incorrect annotation", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			let paths: [PublicPath] = authAccount.privatePaths
		`,
		)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("publicAccount annotation", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			let paths: [PublicPath] = publicAccount.publicPaths
		`,
		)

		require.NoError(t, err)
	})

	t.Run("publicAccount supertype annotation", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			let paths: [Path] = publicAccount.publicPaths
		`,
		)

		require.NoError(t, err)
	})

	t.Run("publicAccount iteration", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				let paths = publicAccount.publicPaths
				for path in paths {
					let cap = publicAccount.getCapability(path)
				}
			}
		`,
		)

		require.NoError(t, err)
	})

	t.Run("iteration type mismatch", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				let paths = authAccount.publicPaths
				for path in paths {
					let t = authAccount.type(at: path)
				}
			}
		`,
		)

		errors := RequireCheckerErrors(t, err, 1)

		// `type` expects a `StoragePath`, not a `PublicPath`
		var mismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &mismatchError)

		assert.Equal(t, "StoragePath", mismatchError.ExpectedType.QualifiedString())
	})

	t.Run("iteration", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				let paths = authAccount.storagePaths
				for storagePath in paths {
					let t = authAccount.type(at: storagePath)
				}
			}
		`,
		)

		require.NoError(t, err)
	})
}

func TestCheckPublicAccountIteration(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				publicAccount.forEachPublic(fun (path: PublicPath, type:Type): Bool {
					return true
				})
			}
			`,
		)

		require.NoError(t, err)
	})

	t.Run("labels irrelevant", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				publicAccount.forEachPublic(fun (foo: PublicPath, bar:Type): Bool {
					return true
				})
			}
			`,
		)

		require.NoError(t, err)
	})

	t.Run("incompatible return", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				publicAccount.forEachPublic(fun (path: PublicPath, type:Type): Bool {
					return 3
				})
			}
			`,
		)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("incompatible return annot", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				publicAccount.forEachPublic(fun (path: PublicPath, type:Type): Void {})
			}
			`,
		)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("incompatible arg 1", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				publicAccount.forEachPublic(fun (path: StoragePath, type:Type): Void {})
			}
			`,
		)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("incompatible arg 2", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				publicAccount.forEachPublic(fun (path: PublicPath, type:Int): Void {})
			}
			`,
		)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("supertype", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t,
			`
			fun test() {
				publicAccount.forEachPublic(fun (path: CapabilityPath, type:Type): Void {})
			}
			`,
		)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})
}

func TestCheckAuthAccountIteration(t *testing.T) {

	t.Parallel()

	t.Run("basic suite", func(t *testing.T) {
		t.Parallel()

		nameTypePairs := []struct {
			name        string
			correctType string
		}{
			{name: "forEachPublic", correctType: "PublicPath"},
			{name: "forEachPrivate", correctType: "PrivatePath"},
			{name: "forEachStored", correctType: "StoragePath"},
		}

		test := func(pair struct {
			name        string
			correctType string
		}) {
			t.Run(fmt.Sprintf("basic %s", pair.correctType), func(t *testing.T) {
				t.Parallel()
				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(`
					fun test() {
						authAccount.%s(fun (path: %s, type:Type): Bool {
							return true
						})
					}
					`, pair.name, pair.correctType),
				)

				require.NoError(t, err)
			})

			t.Run(fmt.Sprintf("labels irrelevant %s", pair.correctType), func(t *testing.T) {
				t.Parallel()
				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(`
					fun test() {
						authAccount.%s(fun (foo: %s, bar:Type): Bool {
							return true
						})
					}
					`, pair.name, pair.correctType),
				)

				require.NoError(t, err)
			})

			t.Run(fmt.Sprintf("incompatible return %s", pair.correctType), func(t *testing.T) {
				t.Parallel()
				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(`
					fun test() {
						authAccount.%s(fun (path: %s, type:Type): Bool {
							return 3
						})
					}
					`, pair.name, pair.correctType),
				)

				errors := RequireCheckerErrors(t, err, 1)
				require.IsType(t, &sema.TypeMismatchError{}, errors[0])
			})

			t.Run(fmt.Sprintf("incompatible return annot %s", pair.correctType), func(t *testing.T) {
				t.Parallel()
				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(`
					fun test() {
						authAccount.%s(fun (path: %s, type:Type): Void {})
					}
					`, pair.name, pair.correctType),
				)

				errors := RequireCheckerErrors(t, err, 1)
				require.IsType(t, &sema.TypeMismatchError{}, errors[0])
			})

			t.Run(fmt.Sprintf("incompatible arg 1 %s", pair.correctType), func(t *testing.T) {
				t.Parallel()
				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(`
					fun test() {
						authAccount.%s(fun (path: Int, type:Type): Void {})
					}
					`, pair.name),
				)

				errors := RequireCheckerErrors(t, err, 1)
				require.IsType(t, &sema.TypeMismatchError{}, errors[0])
			})

			t.Run(fmt.Sprintf("incompatible arg 2 %s", pair.correctType), func(t *testing.T) {
				t.Parallel()
				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(`
					fun test() {
						authAccount.%s(fun (path: %s, type:Int): Void {})
					}
					`, pair.name, pair.correctType),
				)

				errors := RequireCheckerErrors(t, err, 1)
				require.IsType(t, &sema.TypeMismatchError{}, errors[0])
			})

			t.Run(fmt.Sprintf("supertype %s", pair.correctType), func(t *testing.T) {
				t.Parallel()
				_, err := ParseAndCheckAccount(t,
					fmt.Sprintf(`
					fun test() {
						authAccount.%s(fun (path: Path, type:Type): Void {})
					}
					`, pair.name),
				)

				errors := RequireCheckerErrors(t, err, 1)
				require.IsType(t, &sema.TypeMismatchError{}, errors[0])
			})
		}

		for _, pair := range nameTypePairs {
			test(pair)
		}
	})
}

func TestCheckAccountPublish(t *testing.T) {

	t.Parallel()

	t.Run("basic publish", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test(_ cap: Capability<&Int>) {
				let x: Void = authAccount.inbox.publish(cap, name: "foo", recipient: 0x1)
			}`,
		)
		require.NoError(t, err)
	})

	t.Run("publish unlabeled name", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test(_ cap: Capability<&Int>) {
				authAccount.inbox.publish(cap, "foo", recipient: 0x1)
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.MissingArgumentLabelError{}, errors[0])
	})

	t.Run("publish unlabeled recipient", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test(_ cap: Capability<&Int>) {
				authAccount.inbox.publish(cap, name: "foo", 0x1)
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.MissingArgumentLabelError{}, errors[0])
	})

	t.Run("publish wrong argument types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test() {
				authAccount.inbox.publish(3, name: 3, recipient: "")
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 3)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		require.IsType(t, &sema.TypeMismatchError{}, errors[1])
		require.IsType(t, &sema.TypeMismatchError{}, errors[2])
	})

	t.Run("publish non-capability", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test() {
				authAccount.inbox.publish(fun () {}, name: "foo", recipient: 0x1)
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})
}

func TestCheckAccountUnpublish(t *testing.T) {

	t.Parallel()

	t.Run("basic unpublish", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test() {
				let x: Capability<&Int> = authAccount.inbox.unpublish<&Int>("foo")!
			}`,
		)
		require.NoError(t, err)
	})

	t.Run("unpublish wrong argument types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test() {
				authAccount.inbox.unpublish<&String>(4)
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("unpublish wrong return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`
			resource R {}
			fun test() {
				let x <- authAccount.inbox.unpublish<&R>("foo")
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.IncorrectTransferOperationError{}, errors[0])
	})

	t.Run("missing type params", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`
			resource R {}
			fun test() {
				let x = authAccount.inbox.unpublish("foo")!
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errors[0])
	})
}

func TestCheckAccountClaim(t *testing.T) {

	t.Parallel()

	t.Run("basic claim", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test() {
				let x: Capability<&Int> = authAccount.inbox.claim<&Int>("foo", provider: 0x1)!
			}`,
		)
		require.NoError(t, err)
	})

	t.Run("claim wrong argument types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test() {
				authAccount.inbox.claim<&String>(4, provider: "foo")
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 2)
		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		require.IsType(t, &sema.TypeMismatchError{}, errors[1])
	})

	t.Run("claim no provider label", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`fun test() {
				authAccount.inbox.claim<&Int>("foo", 0x1)
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.MissingArgumentLabelError{}, errors[0])
	})

	t.Run("claim wrong return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`
			resource R {}
			fun test() {
				let x <- authAccount.inbox.claim<&R>("foo", provider: 0x1)!
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.IncorrectTransferOperationError{}, errors[0])
	})

	t.Run("claim no type argument", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckAccount(t,
			`
			resource R {}
			fun test() {
				authAccount.inbox.claim("foo", provider: 0x1)
			}`,
		)
		require.Error(t, err)
		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errors[0])
	})
}
