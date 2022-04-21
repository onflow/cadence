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

	return ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues([]sema.ValueDeclaration{
					constantDeclaration("authAccount", sema.AuthAccountType),
					constantDeclaration("publicAccount", sema.PublicAccountType),
				}),
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
				errs := ExpectCheckerErrors(t, err, 1)

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
				errs := ExpectCheckerErrors(t, err, 1)

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
				errs := ExpectCheckerErrors(t, err, 1)

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
				errs := ExpectCheckerErrors(t, err, 1)

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

				errs := ExpectCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
			} else {
				errs := ExpectCheckerErrors(t, err, 3)

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

				errs := ExpectCheckerErrors(t, err, 2)

				require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
				require.IsType(t, &sema.TypeMismatchError{}, errs[1])
			} else {
				errs := ExpectCheckerErrors(t, err, 3)

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
                          authAccount.save<((): Int)>(one, to: /%s/one)
                      }
                    `,
					domainIdentifier,
				),
			)

			if domain == common.PathDomainStorage {
				errs := ExpectCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			} else {
				errs := ExpectCheckerErrors(t, err, 2)

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
				errs := ExpectCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeMismatchError{}, errs[0])
			} else {
				errs := ExpectCheckerErrors(t, err, 2)

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
				errs := ExpectCheckerErrors(t, err, 1)

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
				errs := ExpectCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

			} else {
				errs := ExpectCheckerErrors(t, err, 2)

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
					errs := ExpectCheckerErrors(t, err, 1)

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
					errs := ExpectCheckerErrors(t, err, 1)

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
				errs := ExpectCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

			} else {
				errs := ExpectCheckerErrors(t, err, 2)

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
					errs := ExpectCheckerErrors(t, err, 1)

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
					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

				} else {
					errs := ExpectCheckerErrors(t, err, 2)

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
					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

				} else {
					errs := ExpectCheckerErrors(t, err, 2)

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
					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

				} else {
					errs := ExpectCheckerErrors(t, err, 2)

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
					errs := ExpectCheckerErrors(t, err, 1)

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
					errs := ExpectCheckerErrors(t, err, 1)

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

					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					errs := ExpectCheckerErrors(t, err, 2)

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

					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					errs := ExpectCheckerErrors(t, err, 2)

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
				errs := ExpectCheckerErrors(t, err, 1)

				require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])

			default:
				errs := ExpectCheckerErrors(t, err, 2)

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
					errs := ExpectCheckerErrors(t, err, 1)

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
					errs := ExpectCheckerErrors(t, err, 1)

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
					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

				default:
					errs := ExpectCheckerErrors(t, err, 2)

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
					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

				default:
					errs := ExpectCheckerErrors(t, err, 2)

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
				errs := ExpectCheckerErrors(t, err, 1)

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
				errs := ExpectCheckerErrors(t, err, 1)

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
					errs := ExpectCheckerErrors(t, err, 1)

					require.IsType(t, &sema.TypeMismatchError{}, errs[0])

					return
				} else {
					require.NoError(t, err)
				}

			case common.PathDomainPublic:
				require.NoError(t, err)

			default:
				errs := ExpectCheckerErrors(t, err, 1)

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
		_, err := ParseAndCheckAccount(t, `
          let contracts: AuthAccount.Contracts = authAccount.contracts
	    `)

		require.NoError(t, err)
	})

	t.Run("contracts names", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
          let names: [String] = authAccount.contracts.names
	    `)

		require.NoError(t, err)
	})

	t.Run("update contracts names", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
            fun test() {
                authAccount.contracts.names = ["foo"]
            }
	    `)

		require.Error(t, err)
		errors := ExpectCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errors[0])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errors[1])
	})
}

func TestPublicAccountContracts(t *testing.T) {

	t.Parallel()

	t.Run("contracts type", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
            let contracts: PublicAccount.Contracts = publicAccount.contracts
	    `)

		require.NoError(t, err)
	})

	t.Run("contracts names", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
            let names: [String] = publicAccount.contracts.names
	    `)

		require.NoError(t, err)
	})

	t.Run("update contracts names", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
            fun test() {
                publicAccount.contracts.names = ["foo"]
            }
	    `)

		require.Error(t, err)
		errors := ExpectCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errors[0])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errors[1])
	})

	t.Run("add contract", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
            fun test() {
                publicAccount.contracts.add(name: "foo", code: "012".decodeHex())
            }
	    `)

		require.Error(t, err)
		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredMemberError{}, errors[0])
		notDeclaredError := errors[0].(*sema.NotDeclaredMemberError)
		assert.Equal(t, "add", notDeclaredError.Name)
	})

	t.Run("update contract", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
            fun test() {
                publicAccount.contracts.update__experimental(name: "foo", code: "012".decodeHex())
            }
	    `)

		require.Error(t, err)
		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredMemberError{}, errors[0])
		notDeclaredError := errors[0].(*sema.NotDeclaredMemberError)
		assert.Equal(t, "update__experimental", notDeclaredError.Name)
	})

	t.Run("remove contract", func(t *testing.T) {
		_, err := ParseAndCheckAccount(t, `
            fun test() {
                publicAccount.contracts.remove(name: "foo")
            }
	    `)

		require.Error(t, err)
		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.NotDeclaredMemberError{}, errors[0])
		notDeclaredError := errors[0].(*sema.NotDeclaredMemberError)
		assert.Equal(t, "remove", notDeclaredError.Name)
	})

}
