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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func TestInterpretPath(t *testing.T) {

	t.Parallel()

	for _, domain := range common.AllPathDomainsByIdentifier {

		t.Run(fmt.Sprintf("valid: %s", domain.Identifier()), func(t *testing.T) {

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      let x = /%s/random
                    `,
					domain.Identifier(),
				),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.PathValue{
					Domain:     domain,
					Identifier: "random",
				},
				inter.GetGlobal("x"),
			)
		})
	}
}

func TestInterpretConvertStringToPath(t *testing.T) {
	t.Parallel()

	domainTypes := map[common.PathDomain]sema.Type{
		common.PathDomainStorage: sema.StoragePathType,
		common.PathDomainPublic:  sema.PublicPathType,
		common.PathDomainPrivate: sema.PrivatePathType,
	}

	test := func(domain common.PathDomain) {

		t.Run(fmt.Sprintf("valid: %s", domain.Identifier()), func(t *testing.T) {

			t.Parallel()

			domainType := domainTypes[domain]

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      let x = %[1]s(identifier: "foo")!
                    `,
					domainType.String(),
				),
			)

			assert.Equal(t,
				interpreter.PathValue{
					Domain:     domain,
					Identifier: "foo",
				},
				inter.GetGlobal("x"),
			)
		})

		t.Run(fmt.Sprintf("syntactically invalid identifier 2: %s", domain.Identifier()), func(t *testing.T) {

			t.Parallel()

			domainType := domainTypes[domain]

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      let x = %[1]s(identifier: "2")!
                    `,
					domainType.String(),
				),
			)

			assert.Equal(t,
				interpreter.PathValue{
					Domain:     domain,
					Identifier: "2",
				},
				inter.GetGlobal("x"),
			)
		})

		t.Run(fmt.Sprintf("syntactically invalid identifier -: %s", domain.Identifier()), func(t *testing.T) {

			t.Parallel()

			domainType := domainTypes[domain]

			inter := parseCheckAndPrepare(t,
				fmt.Sprintf(
					`
                      let x = %[1]s(identifier: "fo-o")!
                    `,
					domainType.String(),
				),
			)

			assert.Equal(t,
				interpreter.PathValue{
					Domain:     domain,
					Identifier: "fo-o",
				},
				inter.GetGlobal("x"),
			)
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		test(domain)
	}
}
