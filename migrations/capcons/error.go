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

package capcons

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

// CyclicLinkError
type CyclicLinkError struct {
	Paths   []interpreter.PathValue
	Address common.Address
}

var _ errors.UserError = CyclicLinkError{}

func (CyclicLinkError) IsUserError() {}

func (e CyclicLinkError) Error() string {
	var builder strings.Builder
	for i, path := range e.Paths {
		if i > 0 {
			builder.WriteString(" -> ")
		}
		builder.WriteString(path.String())
	}
	paths := builder.String()

	return fmt.Sprintf(
		"cyclic link in account %s: %s",
		e.Address.ShortHexWithPrefix(),
		paths,
	)
}
