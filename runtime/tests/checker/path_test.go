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
)

func TestCheckPath(t *testing.T) {

	t.Parallel()

	domainTypes := map[common.PathDomain]sema.Type{
		common.PathDomainStorage: sema.StoragePathType,
		common.PathDomainPublic:  sema.PublicPathType,
		common.PathDomainPrivate: sema.PrivatePathType,
	}

	test := func(domain common.PathDomain) {

		t.Run(fmt.Sprintf("valid: %s", domain.Identifier()), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x: Path = /%[1]s/foo
                      let y = /%[1]s/bar
                    `,
					domain.Identifier(),
				),
			)

			require.NoError(t, err)

			assert.IsType(t,
				domainTypes[domain],
				RequireGlobalValue(t, checker.Elaboration, "x"),
			)
		})
	}

	testPathToString := func(domain common.PathDomain) {

		t.Run(fmt.Sprintf("toString: %s", domain.Identifier()), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x = /%[1]s/foo
                      let y = x.toString()
                    `,
					domain.Identifier(),
				),
			)

			require.NoError(t, err)

			assert.IsType(t,
				sema.StringType,
				RequireGlobalValue(t, checker.Elaboration, "y"),
			)
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		test(domain)
		testPathToString(domain)
	}

	t.Run("invalid: unsupported domain", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          let x = /wrong/random
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidPathDomainError{}, errs[0])
	})
}

func TestCheckConvertStringToPath(t *testing.T) {
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

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x = %[1]s(identifier: "foo")
                    `,
					domainType.String(),
				),
			)

			require.NoError(t, err)

			assert.IsType(t,
				&sema.OptionalType{Type: domainTypes[domain]},
				RequireGlobalValue(t, checker.Elaboration, "x"),
			)
		})

		t.Run(fmt.Sprintf("missing argument label: %s", domain.Identifier()), func(t *testing.T) {

			t.Parallel()

			domainType := domainTypes[domain]

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x = %[1]s("foo")
                    `,
					domainType.String(),
				),
			)

			require.IsType(t, &sema.MissingArgumentLabelError{}, RequireCheckerErrors(t, err, 1)[0])
		})
	}

	for _, domain := range common.AllPathDomainsByIdentifier {
		test(domain)
	}
}
