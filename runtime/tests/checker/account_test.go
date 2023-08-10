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
	"github.com/onflow/cadence/runtime/tests/utils"
)

func ParseAndCheckAccountWithConfig(t *testing.T, code string, config sema.Config) (*sema.Checker, error) {

	constantDeclaration := func(name string, ty sema.Type) stdlib.StandardLibraryValue {
		return stdlib.StandardLibraryValue{
			Name: name,
			Type: ty,
			Kind: common.DeclarationKindConstant,
		}
	}

	baseValueActivation := config.BaseValueActivation
	if baseValueActivation == nil {
		baseValueActivation = sema.BaseValueActivation
	}

	baseValueActivation = sema.NewVariableActivation(baseValueActivation)
	baseValueActivation.DeclareValue(constantDeclaration("authAccount", sema.FullyEntitledAccountReferenceType))
	baseValueActivation.DeclareValue(constantDeclaration("publicAccount", sema.AccountReferenceType))
	config.BaseValueActivation = baseValueActivation

	return ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Config: &config,
		},
	)
}

func ParseAndCheckAccount(t *testing.T, code string) (*sema.Checker, error) {
	return ParseAndCheckAccountWithConfig(t, code, sema.Config{})
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
                          authAccount.storage.save(<-r, to: /%s/r)
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
                          authAccount.storage.save(s, to: /%s/s)
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
                          authAccount.storage.save<@R>(<-r, to: /%s/r)
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
                          authAccount.storage.save<S>(s, to: /%s/s)
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
                          authAccount.storage.save<@T>(<-r, to: /%s/r)
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
                          authAccount.storage.save<T>(s, to: /%s/s)
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
                          authAccount.storage.save<fun(): Int>(one, to: /%s/one)
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
                          authAccount.storage.save(one, to: /%s/one)
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
						let t: Type = authAccount.storage.type(at: /%s/r)!
					`,
					domain.Identifier(),
				),
			)

			if domain == common.PathDomainStorage {

				require.NoError(t, err)

				typ := RequireGlobalValue(t, checker.Elaboration, "t")

				require.Equal(t, sema.MetaType, typ)

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
                      let s = authAccount.storage.load(from: /%s/s)
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

                          let r <- authAccount.storage.load<@R>(from: /%s/r)
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

                          let s = authAccount.storage.load<S>(from: /%s/s)
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

                      let s = authAccount.storage.copy(from: /%s/s)
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

                          let s = authAccount.storage.copy<S>(from: /%s/s)
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

                          let r <- authAccount.storage.copy<@R>(from: /%s/r)
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
                          let r = authAccount.storage.borrow(from: /%s/r)
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
                          let s = authAccount.storage.borrow(from: /%s/s)
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

	testExplicitTypeArgumentReference := func(domain common.PathDomain, auth sema.Access) {

		authKeyword := auth.AuthKeyword()

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
						  entitlement X

                          let r = authAccount.storage.borrow<%s &R>(from: /%s/r)
                        `,
						authKeyword,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {

					require.NoError(t, err)

					rType := RequireGlobalType(t, checker.Elaboration, "R")
					rValueType := RequireGlobalValue(t, checker.Elaboration, "r")

					xType := RequireGlobalType(t, checker.Elaboration, "X")
					require.IsType(t, &sema.EntitlementType{}, xType)
					xEntitlement := xType.(*sema.EntitlementType)
					var access sema.Access = sema.UnauthorizedAccess
					if !auth.Equal(sema.UnauthorizedAccess) {
						access = sema.NewEntitlementSetAccess([]*sema.EntitlementType{xEntitlement}, sema.Conjunction)
					}

					require.Equal(t,
						&sema.OptionalType{
							Type: &sema.ReferenceType{
								Authorization: access,
								Type:          rType,
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
						  entitlement X

                          let s = authAccount.storage.borrow<%s &S>(from: /%s/s)
                        `,
						authKeyword,
						domain.Identifier(),
					),
				)

				if domain == common.PathDomainStorage {
					require.NoError(t, err)

					sType := RequireGlobalType(t, checker.Elaboration, "S")
					sValueType := RequireGlobalValue(t, checker.Elaboration, "s")

					xType := RequireGlobalType(t, checker.Elaboration, "X")
					require.IsType(t, &sema.EntitlementType{}, xType)
					xEntitlement := xType.(*sema.EntitlementType)
					var access sema.Access = sema.UnauthorizedAccess
					if !auth.Equal(sema.UnauthorizedAccess) {
						access = sema.NewEntitlementSetAccess([]*sema.EntitlementType{xEntitlement}, sema.Conjunction)
					}

					require.Equal(t,
						&sema.OptionalType{
							Type: &sema.ReferenceType{
								Authorization: access,
								Type:          sType,
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

                          let r <- authAccount.storage.borrow<@R>(from: /%s/r)
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

                          let s = authAccount.storage.borrow<S>(from: /%s/s)
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

		for _, auth := range []sema.Access{
			sema.UnauthorizedAccess,
			sema.NewEntitlementSetAccess(
				[]*sema.EntitlementType{
					{
						Location:   utils.TestLocation,
						Identifier: "X",
					},
				},
				sema.Conjunction),
		} {
			testExplicitTypeArgumentReference(domain, auth)
		}

		testExplicitTypeArgumentNonReference(domain)
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
			"storage.used",
			"storage.capacity",
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
          let contracts: &Account.Contracts = authAccount.contracts
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
            fun test(): DeployedContract {
                return authAccount.contracts.get(name: "foo")!
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("borrow contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &C {
                return authAccount.contracts.borrow<&C>(name: "foo")!
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("invalid borrow contract: missing type argument", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &AnyStruct {
                return authAccount.contracts.borrow(name: "foo")!
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
                return authAccount.contracts.update(name: "foo", code: "012".decodeHex())
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("remove contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract {
                return authAccount.contracts.remove(name: "foo")!
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
            let contracts: &Account.Contracts = publicAccount.contracts
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
            fun test(): DeployedContract {
                return publicAccount.contracts.get(name: "foo")!
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("borrow contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &C {
                return publicAccount.contracts.borrow<&C>(name: "foo")!
            }
	    `)

		require.NoError(t, err)
	})

	t.Run("invalid borrow contract: missing type argument", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            contract C {}

            fun test(): &AnyStruct {
                return publicAccount.contracts.borrow(name: "foo")!
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

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errors[0], &invalidAccessErr)
		assert.Equal(t, "add", invalidAccessErr.Name)
	})

	t.Run("update contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract {
                return publicAccount.contracts.update(name: "foo", code: "012".decodeHex())
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errors[0], &invalidAccessErr)
		assert.Equal(t, "update", invalidAccessErr.Name)
	})

	t.Run("remove contract", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAndCheckAccount(t, `
            fun test(): DeployedContract? {
                return publicAccount.contracts.remove(name: "foo")
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		var invalidAccessErr *sema.InvalidAccessError
		require.ErrorAs(t, errors[0], &invalidAccessErr)
		assert.Equal(t, "remove", invalidAccessErr.Name)
	})

}

func TestCheckAccountStoragePaths(t *testing.T) {

	t.Parallel()

	t.Run("capitalized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun test(storage: &Account.Storage) {
                let paths = storage.StoragePaths
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		var notDeclaredError *sema.NotDeclaredMemberError
		require.ErrorAs(t, errors[0], &notDeclaredError)

		assert.Equal(t, "StoragePaths", notDeclaredError.Name)
	})

	t.Run("annotation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(storage: &Account.Storage) {
              let publicPaths: &[PublicPath] = storage.publicPaths
              let storagePaths: &[StoragePath] = storage.storagePaths
          }
        `)
		require.NoError(t, err)
	})

	t.Run("supertype annotation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(storage: &Account.Storage) {
              let publicPaths: &[Path] = storage.publicPaths
              let storagePaths: &[Path] = storage.storagePaths
          }
        `)
		require.NoError(t, err)
	})

	t.Run("incorrect annotation", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(storage: &Account.Storage) {
              let paths: &[PublicPath] = storage.storagePaths
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})
}

func TestCheckAccountStorageIteration(t *testing.T) {

	t.Parallel()

	type testCase struct {
		storageRefType string
		functionName   string
		pathType       string
	}

	test := func(t *testing.T, testCase testCase) {
		t.Run(fmt.Sprintf("basic %s", testCase.pathType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                          fun test(storage: %s) {
                              storage.%s(fun (path: %s, type: Type): Bool {
                                  return true
                              })
                          }
                        `,
					testCase.storageRefType,
					testCase.functionName,
					testCase.pathType,
				),
			)
			require.NoError(t, err)
		})

		t.Run(fmt.Sprintf("labels irrelevant %s", testCase.pathType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                          fun test(storage: %s) {
                              storage.%s(fun (foo: %s, bar: Type): Bool {
                                  return true
                              })
                          }
                        `,
					testCase.storageRefType,
					testCase.functionName,
					testCase.pathType,
				),
			)
			require.NoError(t, err)
		})

		t.Run(fmt.Sprintf("incompatible return %s", testCase.pathType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                          fun test(storage: %s) {
                              storage.%s(fun (path: %s, type: Type): Bool {
                                  return 3
                              })
                          }
                        `,
					testCase.storageRefType,
					testCase.functionName,
					testCase.pathType,
				),
			)

			errors := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		})

		t.Run(fmt.Sprintf("incompatible return annotation %s", testCase.pathType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                          fun test(storage: %s) {
                              storage.%s(fun (path: %s, type: Type): Void {})
                          }
                        `,
					testCase.storageRefType,
					testCase.functionName,
					testCase.pathType,
				),
			)

			errors := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		})

		t.Run(fmt.Sprintf("incompatible arg 1 %s", testCase.pathType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                          fun test(storage: %s) {
                              storage.%s(fun (path: Int, type: Type): Void {})
                          }
                        `,
					testCase.storageRefType,
					testCase.functionName,
				),
			)

			errors := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		})

		t.Run(fmt.Sprintf("incompatible arg 2 %s", testCase.pathType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                          fun test(storage: %s) {
                              storage.%s(fun (path: %s, type: Int): Void {})
                          }
                        `,
					testCase.storageRefType,
					testCase.functionName,
					testCase.pathType,
				),
			)

			errors := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		})

		t.Run(fmt.Sprintf("supertype %s", testCase.pathType), func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                          fun test(storage: %s) {
                              storage.%s(fun (path: Path, type: Type): Void {})
                          }
                        `,
					testCase.storageRefType,
					testCase.functionName,
				),
			)

			errors := RequireCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		})
	}

	functionPairs := []struct {
		functionName string
		pathType     string
	}{
		{functionName: "forEachPublic", pathType: "PublicPath"},
		{functionName: "forEachStored", pathType: "StoragePath"},
	}

	for _, storageRefType := range []string{
		"auth(Storage) &Account.Storage",
		"&Account.Storage",
	} {
		t.Run(storageRefType, func(t *testing.T) {

			for _, pair := range functionPairs {
				test(t, testCase{
					storageRefType: storageRefType,
					functionName:   pair.functionName,
					pathType:       pair.pathType,
				})
			}
		})
	}
}

func TestCheckAccountInboxPublish(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(cap: Capability<&Int>, inbox: auth(Inbox) &Account.Inbox) {
              let x: Void = inbox.publish(cap, name: "foo", recipient: 0x1)
          }
        `)
		require.NoError(t, err)
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(cap: Capability<&Int>, inbox: &Account.Inbox) {
              inbox.publish(cap, name: "foo", recipient: 0x1)
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidAccessError{}, errors[0])
	})

	t.Run("unlabeled name", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(cap: Capability<&Int>, inbox: auth(Inbox) &Account.Inbox) {
              inbox.publish(cap, "foo", recipient: 0x1)
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.MissingArgumentLabelError{}, errors[0])
	})

	t.Run("unlabeled recipient", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(cap: Capability<&Int>, inbox: auth(Inbox) &Account.Inbox) {
              inbox.publish(cap, name: "foo", 0x1)
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.MissingArgumentLabelError{}, errors[0])
	})

	t.Run("wrong argument types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: auth(Inbox) &Account.Inbox) {
              inbox.publish(3, name: 3, recipient: "")
          }
		`)

		errors := RequireCheckerErrors(t, err, 3)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		require.IsType(t, &sema.TypeMismatchError{}, errors[1])
		require.IsType(t, &sema.TypeMismatchError{}, errors[2])
	})

	t.Run("non-capability", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: auth(Inbox) &Account.Inbox) {
              inbox.publish(fun () {}, name: "foo", recipient: 0x1)
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})
}

func TestCheckAccountInboxUnpublish(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: auth(Inbox) &Account.Inbox) {
              let x: Capability<&Int> = inbox.unpublish<&Int>("foo")!
          }
        `)
		require.NoError(t, err)
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: &Account.Inbox) {
              inbox.unpublish<&Int>("foo")
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidAccessError{}, errors[0])
	})

	t.Run("wrong argument types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: auth(Inbox) &Account.Inbox) {
              inbox.unpublish<&String>(4)
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("wrong return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
          resource R {}

          fun test(inbox: auth(Inbox) &Account.Inbox) {
              let x <- inbox.unpublish<&R>("foo")
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.IncorrectTransferOperationError{}, errors[0])
	})

	t.Run("missing type params", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(inbox: auth(Inbox) &Account.Inbox) {
              let x = inbox.unpublish("foo")!
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errors[0])
	})
}

func TestCheckAccountInboxClaim(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: auth(Inbox) &Account.Inbox) {
              let x: Capability<&Int> = inbox.claim<&Int>("foo", provider: 0x1)!
          }
        `)
		require.NoError(t, err)
	})

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: &Account.Inbox) {
              let x: Capability<&Int> = inbox.claim<&Int>("foo", provider: 0x1)!
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidAccessError{}, errors[0])
	})

	t.Run("wrong argument types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: auth(Inbox) &Account.Inbox) {
              inbox.claim<&String>(4, provider: "foo")
          }
        `)

		errors := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		require.IsType(t, &sema.TypeMismatchError{}, errors[1])
	})

	t.Run("no provider label", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(inbox: auth(Inbox) &Account.Inbox) {
              inbox.claim<&Int>("foo", 0x1)
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.MissingArgumentLabelError{}, errors[0])
	})

	t.Run("wrong return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(inbox: auth(Inbox) &Account.Inbox) {
              let x <- inbox.claim<&R>("foo", provider: 0x1)!
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.IncorrectTransferOperationError{}, errors[0])
	})

	t.Run("no type argument", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(inbox: auth(Inbox) &Account.Inbox) {
              inbox.claim("foo", provider: 0x1)
          }
        `)

		errors := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errors[0])
	})
}

func TestCheckAccountCapabilities(t *testing.T) {

	t.Parallel()

	t.Run("no authorization required", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(capabilities: &Account.Capabilities) {

              let cap: Capability<&Int> = capabilities.get<&Int>(/public/foo)!

              let ref: &Int = capabilities.borrow<&Int>(/public/foo)!
          }
        `)
		require.NoError(t, err)
	})

	t.Run("with authorization", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(capabilities: auth(Capabilities) &Account.Capabilities) {

              let cap: Capability<&Int> = capabilities.get<&Int>(/public/foo)!

              let ref: &Int = capabilities.borrow<&Int>(/public/foo)!

              capabilities.publish(cap, at: /public/bar)

              let cap2: Capability = capabilities.unpublish(/public/bar)!
          }
        `)
		require.NoError(t, err)
	})

	t.Run("without authorization", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
           fun test(capabilities: &Account.Capabilities) {

              let cap: Capability<&Int> = capabilities.get<&Int>(/public/foo)!

              capabilities.publish(cap, at: /public/bar)

              let cap2: Capability = capabilities.unpublish(/public/bar)!
          }
        `)

		errors := RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidAccessError{}, errors[0])
		require.IsType(t, &sema.InvalidAccessError{}, errors[1])
	})
}

func TestCheckAccountStorageCapabilities(t *testing.T) {

	t.Parallel()

	t.Run("with authorization", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(capabilities: auth(StorageCapabilities) &Account.StorageCapabilities) {

              let controller: &StorageCapabilityController = capabilities.getController(byCapabilityID: 1)!

              let controllers: [&StorageCapabilityController] = capabilities.getControllers(forPath: /storage/foo)

              capabilities.forEachController(
                  forPath: /storage/bar,
                  fun (controller: &StorageCapabilityController): Bool {
                      return true
                  }
              )

              let cap2: Capability<&String> = capabilities.issue<&String>(/storage/baz)
          }
        `)
		require.NoError(t, err)
	})

	t.Run("without authorization", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(capabilities: &Account.StorageCapabilities) {

              let controller: &StorageCapabilityController = capabilities.getController(byCapabilityID: 1)!

              let controllers: [&StorageCapabilityController] = capabilities.getControllers(forPath: /storage/foo)

              capabilities.forEachController(
                  forPath: /storage/bar,
                  fun (controller: &StorageCapabilityController): Bool {
                      return true
                  }
              )

              let cap2: Capability<&String> = capabilities.issue<&String>(/storage/baz)
          }
        `)

		errors := RequireCheckerErrors(t, err, 4)

		require.IsType(t, &sema.InvalidAccessError{}, errors[0])
		require.IsType(t, &sema.InvalidAccessError{}, errors[1])
		require.IsType(t, &sema.InvalidAccessError{}, errors[2])
		require.IsType(t, &sema.InvalidAccessError{}, errors[3])
	})
}

func TestCheckAccountAccountCapabilities(t *testing.T) {

	t.Parallel()

	t.Run("with authorization", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(capabilities: auth(AccountCapabilities) &Account.AccountCapabilities) {

              let controller: &AccountCapabilityController = capabilities.getController(byCapabilityID: 1)!

              let controllers: [&AccountCapabilityController] = capabilities.getControllers()

              capabilities.forEachController(fun (controller: &AccountCapabilityController): Bool {
                  return true
              })

              let cap: Capability<&Account> = capabilities.issue<&Account>()
          }
        `)
		require.NoError(t, err)
	})

	t.Run("without authorization", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(capabilities: &Account.AccountCapabilities) {

              let controller: &AccountCapabilityController = capabilities.getController(byCapabilityID: 1)!

              let controllers: [&AccountCapabilityController] = capabilities.getControllers()

              capabilities.forEachController(fun (controller: &AccountCapabilityController): Bool {
                  return true
              })

              let cap: Capability<&Account> = capabilities.issue<&Account>()
          }
        `)

		errors := RequireCheckerErrors(t, err, 4)

		require.IsType(t, &sema.InvalidAccessError{}, errors[0])
		require.IsType(t, &sema.InvalidAccessError{}, errors[1])
		require.IsType(t, &sema.InvalidAccessError{}, errors[2])
		require.IsType(t, &sema.InvalidAccessError{}, errors[3])
	})
}
