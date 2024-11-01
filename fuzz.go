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

package cadence

import (
	"unicode/utf8"

	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func Fuzz(data []byte) int {

	if !utf8.Valid(data) {
		return 0
	}

	program, err := parser.ParseProgram(nil, data, parser.Config{})

	if err != nil {
		return 0
	}

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		nil,
		&sema.Config{
			AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
		},
	)
	if err != nil {
		return 0
	}

	err = checker.Check()
	if err != nil {
		return 0
	}

	return 1
}
