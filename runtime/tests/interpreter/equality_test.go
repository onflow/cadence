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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestInterpretEquality(t *testing.T) {

	t.Parallel()

	t.Run("capability", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpretWithOptions(t,
			`
              let maybeCapNonNil: Capability? = cap
              let maybeCapNil: Capability? = nil
		      let res1 = maybeCapNonNil != nil
		      let res2 = maybeCapNil == nil
		    `,
			ParseCheckAndInterpretOptions{
				Options: []interpreter.Option{
					interpreter.WithPredefinedValues(map[string]interpreter.Value{
						"cap": interpreter.CapabilityValue{
							Address: interpreter.NewAddressValue(common.BytesToAddress([]byte{0x1})),
							Path: interpreter.PathValue{
								Domain:     common.PathDomainStorage,
								Identifier: "something",
							},
						},
					}),
				},
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(map[string]sema.ValueDeclaration{
						"cap": stdlib.StandardLibraryValue{
							Name:       "cap",
							Type:       &sema.CapabilityType{},
							Kind:       common.DeclarationKindConstant,
							IsConstant: true,
						},
					}),
				},
			},
		)

		assert.Equal(t,
			interpreter.BoolValue(true),
			inter.Globals["res1"].Value,
		)

		assert.Equal(t,
			interpreter.BoolValue(true),
			inter.Globals["res2"].Value,
		)
	})

	t.Run("function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		  fun func() {}

          let maybeFuncNonNil: ((): Void)? = func
          let maybeFuncNil: ((): Void)? = nil
		  let res1 = maybeFuncNonNil != nil
		  let res2 = maybeFuncNil == nil
		`)

		assert.Equal(t,
			interpreter.BoolValue(true),
			inter.Globals["res1"].Value,
		)

		assert.Equal(t,
			interpreter.BoolValue(true),
			inter.Globals["res2"].Value,
		)
	})
}
