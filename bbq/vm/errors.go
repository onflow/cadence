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

package vm

import (
	"fmt"

	"github.com/onflow/cadence/errors"
)

type LinkerError struct {
	Message string
}

var _ error = LinkerError{}
var _ errors.InternalError = LinkerError{}

func (l LinkerError) IsInternalError() {
}

func (l LinkerError) Error() string {
	return l.Message
}

// ForceNilError
type ForceNilError struct{}

var _ errors.UserError = ForceNilError{}

func (ForceNilError) IsUserError() {}

func (e ForceNilError) Error() string {
	return "unexpectedly found nil while forcing an Optional value"
}

// UnknownFunctionError
type UnknownFunctionError struct {
	name string
}

var _ errors.UserError = UnknownFunctionError{}

func (UnknownFunctionError) IsUserError() {}

func (e UnknownFunctionError) Error() string {
	return fmt.Sprintf("unknown function `%s`", e.name)
}
