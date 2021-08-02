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
	"fmt"
	"testing"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretPath(t *testing.T) {

	t.Parallel()

	for _, domain := range common.AllPathDomainsByIdentifier {

		t.Run(fmt.Sprintf("valid: %s", domain.Identifier()), func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let x = /%s/random
                    `,
					domain.Identifier(),
				),
			)

			AssertValuesEqual(t,
				interpreter.PathValue{
					Domain:     domain,
					Identifier: "random",
				},
				inter.Globals["x"].GetValue(),
			)
		})
	}
}
