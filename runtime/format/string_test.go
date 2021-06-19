/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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

package format

import (
	"testing"
	"testing/quick"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		`"test xyz \u{1f496}"`,
		String("test xyz \U0001f496"),
	)

	assert.Equal(t,
		`"Foo \u{a9} bar \u{1d306} baz \u{2603} qux"`,
		// "Foo © bar 𝌆 baz ☃ qux"
		String("\x46\x6F\x6F\x20\xC2\xA9\x20\x62\x61\x72\x20\xF0\x9D\x8C\x86\x20\x62\x61\x7A\x20\xE2\x98\x83\x20\x71\x75\x78"),
	)

	assert.Equal(t,
		`"\0"`,
		String("\x00"),
	)

	assert.Equal(t,
		`"\n"`,
		String("\n"),
	)

	assert.Equal(t,
		`"\r"`,
		String("\r"),
	)

	assert.Equal(t,
		`"\t"`,
		String("\t"),
	)

	assert.Equal(t,
		`"\\"`,
		String("\\"),
	)

	assert.Equal(t,
		`"\""`,
		String(`"`),
	)
}

func TestStringQuick(t *testing.T) {
	f := func(text string) bool {
		res, errs := parser2.ParseExpression(String(text))
		if len(errs) > 0 {
			return false
		}
		literal, ok := res.(*ast.StringExpression)
		if !ok {
			return false
		}
		return literal.Value == text
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
