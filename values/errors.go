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

package values

import "github.com/onflow/cadence/errors"

// InvalidOperandsError

type InvalidOperandsError struct{}

var _ errors.UserError = InvalidOperandsError{}

func (InvalidOperandsError) IsUserError() {}

func (InvalidOperandsError) Error() string {
	return "invalid operands"
}

// UnderflowError

type UnderflowError struct{}

var _ errors.UserError = UnderflowError{}

func (UnderflowError) IsUserError() {}

func (UnderflowError) Error() string {
	return "underflow"
}

// OverflowError

type OverflowError struct{}

var _ errors.UserError = OverflowError{}

func (OverflowError) IsUserError() {}

func (OverflowError) Error() string {
	return "overflow"
}

// NegativeShiftError

type NegativeShiftError struct{}

var _ errors.UserError = NegativeShiftError{}

func (NegativeShiftError) IsUserError() {}

func (NegativeShiftError) Error() string {
	return "negative shift"
}

// DivisionByZeroError

type DivisionByZeroError struct{}

var _ errors.UserError = DivisionByZeroError{}

func (DivisionByZeroError) IsUserError() {}

func (DivisionByZeroError) Error() string {
	return "division by zero"
}
