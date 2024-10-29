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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/tests/sema_utils"
)

func TestCheckInvalidCompositeInitializerOverloading(t *testing.T) {

	t.Parallel()

	interfacePossibilities := []bool{true, false}

	for _, kind := range common.CompositeKindsWithFieldsAndFunctions {
		for _, isInterface := range interfacePossibilities {

			if isInterface && !kind.SupportsInterfaces() {
				continue
			}

			interfaceKeyword := ""
			body := ""
			if isInterface {
				interfaceKeyword = "interface"
			} else {
				body = "{}"
			}

			testName := fmt.Sprintf("%s_%s",
				kind.Keyword(),
				interfaceKeyword,
			)

			var baseType string
			if kind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			t.Run(testName, func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          %[1]s %[2]s X %[4]s {
                              init() %[3]s
                              init(y: Int) %[3]s
                          }
                        `,
						kind.Keyword(),
						interfaceKeyword,
						body,
						baseType,
					),
				)

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.RedeclarationError{}, errs[0])
			})
		}
	}
}
