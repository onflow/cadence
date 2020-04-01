package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/sema"
	"github.com/dapperlabs/cadence/runtime/stdlib"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

var accountValueDeclaration = stdlib.StandardLibraryValue{
	Name:       "account",
	Type:       &sema.AuthAccountType{},
	Kind:       common.DeclarationKindConstant,
	IsConstant: true,
}

func ParseAndCheckAccount(t *testing.T, code string) (*sema.Checker, error) {
	return ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(map[string]sema.ValueDeclaration{
					"account": accountValueDeclaration,
				}),
			},
		},
	)
}

func TestCheckAccount(t *testing.T) {

	t.Run("storage is assignable", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          fun test(): @R? {
              let r <- account.storage[R] <- create R()
              return <-r
          }
        `)

		require.NoError(t, err)
	})

	t.Run("published is assignable", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          fun test() {
              account.published[&R] = &account.storage[R] as &R
          }
        `)

		require.NoError(t, err)
	})

	t.Run("saving: implicit type argument", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          fun test() {
              let r <- create R()
              account.save(<-r, to: /storage/r)
          }
        `)

		require.NoError(t, err)
	})

	t.Run("saving: explicit type argument", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          fun test() {
              let r <- create R()
              account.save<@R>(<-r, to: /storage/r)
          }
        `)

		require.NoError(t, err)
	})

	t.Run("saving: explicit type argument, incorrect", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          resource T {}

          fun test() {
              let r <- create R()
              account.save<@T>(<-r, to: /storage/r)
          }
        `)

		errs := ExpectCheckerErrors(t, err, 2)

		require.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		require.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("loading: missing type argument", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          let r <- account.load(from: /storage/r)
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	t.Run("loading: explicit type argument", func(t *testing.T) {

		checker, err := ParseAndCheckAccount(t, `

          resource R {}

          let r <- account.load<@R>(from: /storage/r)
        `)

		require.NoError(t, err)

		rType := checker.GlobalTypes["R"].Type

		rValueType := checker.GlobalValues["r"].Type

		require.Equal(t,
			&sema.OptionalType{
				Type: rType,
			},
			rValueType,
		)
	})

	t.Run("borrowing: missing type argument", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          let r <- account.borrow(from: /storage/r)
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	for _, auth := range []bool{false, true} {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf(
			"borrowing: explicit type argument, %s reference",
			authKeyword,
		)

		t.Run(testName, func(t *testing.T) {

			checker, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      resource R {}

                      let r = account.borrow<%s &R>(from: /storage/r)
                    `,
					authKeyword,
				),
			)

			require.NoError(t, err)

			rType := checker.GlobalTypes["R"].Type

			rValueType := checker.GlobalValues["r"].Type

			require.Equal(t,
				&sema.OptionalType{
					Type: &sema.ReferenceType{
						Authorized: auth,
						Type:       rType,
					},
				},
				rValueType,
			)
		})
	}

	t.Run("borrowing: explicit type argument, non-reference type", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          let r <- account.borrow<@R>(from: /storage/r)
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("linking: missing type argument", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          fun test(): Capability? {
              return account.link(/public/r, target: /storage/r)
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	for _, auth := range []bool{false, true} {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf(
			"linking: explicit type argument, %s reference",
			authKeyword,
		)

		t.Run(testName, func(t *testing.T) {

			_, err := ParseAndCheckAccount(t,
				fmt.Sprintf(
					`
                      resource R {}

                      fun test(): Capability? {
                          return account.link<%s &R>(/public/r, target: /storage/r)
                      }
                    `,
					authKeyword,
				),
			)

			require.NoError(t, err)
		})
	}

	t.Run("linking: explicit type argument, non-reference type", func(t *testing.T) {

		_, err := ParseAndCheckAccount(t, `

          resource R {}

          fun test(): Capability? {
              return account.link<@R>(/public/r, target: /storage/r)
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}
