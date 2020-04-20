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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckPath(t *testing.T) {

	for _, domain := range common.AllPathDomainsByIdentifier {

		t.Run(fmt.Sprintf("valid: %s", domain.Name()), func(t *testing.T) {

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
				&sema.PathType{},
				checker.GlobalValues["x"].Type,
			)
		})
	}

	t.Run("invalid: unsupported domain", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          let x = /wrong/random
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidPathDomainError{}, errs[0])
	})
}
