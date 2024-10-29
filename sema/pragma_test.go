/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package sema_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/tests/sema_utils"
)

func TestCheckPragmaExpression(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name  string
		code  string
		valid bool
	}

	test := func(testCase testCase) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, fmt.Sprintf(`#%s`, testCase.code))

			if testCase.valid {
				require.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)
				assert.IsType(t, &sema.InvalidPragmaError{}, errs[0])
			}
		})
	}

	testCases := []testCase{
		{"string", `"string"`, false},
		{"bool", `true`, false},
		{"integer", `1`, false},
		{"fixed-point", `1.2`, false},
		{"unary, minus", `-1`, false},
		{"unary, move", `<-r`, false},
		{"array", `[1, 2, 3]`, false},
		{"dictionary", `{1: 2}`, false},
		{"nil", `nil`, false},
		{"path", `/storage/foo`, false},
		{"reference", `&x`, false},
		{"index", `xs[0]`, false},
		{"binary", `a + b`, false},
		{"conditional", `a ? b : c`, false},
		{"force", `a!`, false},
		{"function", `fun() {}`, false},
		{"create", `create R()`, false},
		{"destroy", `destroy r`, false},
		{"identifier", `foo`, true},
		// invocations
		{"invocation of member", `foo.bar()`, false},
		{"invocation with type arguments", `foo<X>()`, false},
		{"invocation without arguments", `foo()`, true},
		{"invocation with identifier argument", `foo(bar)`, false},
		{"invocation with string argument", `foo("string")`, true},
		{"invocation with bool argument", `foo(true)`, true},
		{"invocation with integer argument", `foo(1)`, true},
		{"invocation with fixed-point argument", `foo(1.2)`, true},
		{"invocation with unary (minus) argument", `foo(-1)`, true},
		{"invocation with unary (move) argument", `foo(<-r)`, false},
		{"invocation with array argument", `foo([1, 2, 3])`, true},
		{"invocation with dictionary argument", `foo({1: 2})`, true},
		{"invocation with nil argument", `foo(nil)`, true},
		{"invocation with path argument", `foo(/storage/foo)`, true},
		{"invocation with reference argument", `foo(&x)`, false},
		{"invocation with index argument", `foo(xs[0])`, false},
		{"invocation with binary argument", `foo(a + b)`, false},
		{"invocation with conditional argument", `foo(a ? b : c)`, false},
		{"invocation with force argument", `foo(a!)`, false},
		{"invocation with function argument", `foo(fun() {})`, false},
		{"invocation with create argument", `foo(create R())`, false},
		{"invocation with destroy argument", `destroy r`, false},
		// nested invocations
		{"nested invocation without argument", `foo(bar())`, true},
		{"nested invocation with identifier argument", `foo(bar(baz))`, false},
		{"nested invocation with string argument", `foo(bar("string"))`, true},
		// FLIX
		{
			"FLIX",
			`interaction(
                version: "1.1.0",
                title: "Flow Token Balance",
                description: "Get account Flow Token balance",
                language: "en-US",
                parameters: [
                    Parameter(
                        name: "address",
                        title: "Address",
                        description: "Get Flow token balance of Flow account"
                    )
                ],
            )`,
			true,
		},
	}

	for _, testCase := range testCases {
		test(testCase)
	}
}

func TestCheckPragmaInvalidLocation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          #version
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}
