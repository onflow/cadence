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

package runtime

import (
	"fmt"
	"testing"

	"github.com/onflow/cadence/runtime/parser2"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestVariousAgainstRegression(t *testing.T) {
	t.Parallel()

	doTest := func(code string, verify func(t *testing.T, err error)) {
		t.Run(code, func(t *testing.T) {
			script := fmt.Sprintf(`
				pub fun main() {
					%s
				}`, code)
			_, err := executeTestScript(t, script, nil)
			verify(t, err)
		})
	}

	testChildErrors := func(t *testing.T, e error, childErrors []error) {
		e2, ok := e.(interface{ ChildErrors() []error })
		require.Truef(t, ok, "Error does not implement ChildErrors(): %e", e)
		require.Lenf(t, e2.ChildErrors(), len(childErrors),
			"Error has the wrong number of children: %d != %d", len(e2.ChildErrors()), len(childErrors))
		for i, e3 := range childErrors {
			require.IsType(t, childErrors[i], e3)
		}
	}

	doTest("paste your code in here", func(t *testing.T, err error) {
		var e *ParsingCheckingError
		require.ErrorAs(t, err, &e)
	})

	doTest("resource as { enum x : as { } }", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.InvalidDeclarationError{}})
	})

	doTest("resource interface struct{struct d:struct{ struct d:struct{ }struct d:struct{ struct d:struct{ }}}}", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.InvalidDeclarationError{}})
	})

	doTest("resource interface struct{struct d:struct{ contract d:struct{ contract x:struct{ struct d{} contract d:struct{ contract d:struct {}}}}}}", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.InvalidDeclarationError{}})
	})

	doTest("struct interface var { contract h : var { contract h { } contract h { contract h { } } } }", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.InvalidDeclarationError{}})
	})

	doTest("contract signatureAlgorithm { resource interface payer { contract fun : payer { contract fun { contract fun { } contract fun { contract interface account { } } contract account { } } } } }", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.InvalidDeclarationError{}})
	})

	doTest("# a ( b > c > d > e > f > g > h > i > j > k > l > m > n > o > p > q > r > s > t > u > v > w > x > y > z > A > B > C > D > E > F>", func(t *testing.T, err error) {
		var e *ParsingCheckingError
		require.ErrorAs(t, err, &e)

		require.IsType(t, e.Unwrap(), parser2.Error{})
		testChildErrors(t, e.Unwrap().(parser2.Error), []error{&parser2.SyntaxError{}})
	})

	doTest("let b=0.0as!PublicKey.Contracts", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.InvalidDeclarationError{}})
	})

	doTest("let UInt64 = UInt64 ( 0b0 )", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.RedeclarationError{}})
	})

	doTest("contract enum{}let x = enum!", func(t *testing.T, err error) {
		var e *ParsingCheckingError
		require.ErrorAs(t, err, &e)

		require.IsType(t, e.Unwrap(), parser2.Error{})
		testChildErrors(t, e.Unwrap().(parser2.Error), []error{&parser2.SyntaxError{}})
	})

	doTest("#0x0<{},>()", func(t *testing.T, err error) {
		var e *ParsingCheckingError
		require.ErrorAs(t, err, &e)

		require.IsType(t, e.Unwrap(), parser2.Error{})
		testChildErrors(t, e.Unwrap().(parser2.Error), []error{&parser2.SyntaxError{}})
	})

	doTest("var a=[Type]", func(t *testing.T, err error) {
		require.NoError(t, err)
	})

	doTest("var j={0.0:Type}", func(t *testing.T, err error) {
		require.NoError(t, err)
	})

	doTest("let Type = Type", func(t *testing.T, err error) {
		var e *sema.CheckerError
		require.ErrorAs(t, err, &e)
		testChildErrors(t, e, []error{&sema.RedeclarationError{}})
	})

	doTest("let a = 0x0 as UInt64!as?UInt64!as?UInt64?!?.getType()", func(t *testing.T, err error) {
		require.NoError(t, err)
	})
}
