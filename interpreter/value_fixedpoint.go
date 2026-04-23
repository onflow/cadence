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

package interpreter

import (
	fix "github.com/onflow/fixed-point"

	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

func extractRoundingRule(value Value) fix.RoundingMode {
	composite, ok := value.(*SimpleCompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	rawValue, ok := composite.Fields[sema.EnumRawValueFieldName].(UInt8Value)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return fix.RoundingMode(rawValue)
}

// handleFixedPointConversionError handles errors from the fixed-point library
// during narrowing conversions (e.g. Fix128 → Fix64).
//
// Unlike handleFixedpointError (used for Fix128 arithmetic),
// this function does NOT ignore UnderflowError:
// for narrowing conversions, a nonzero value that rounds to zero
// is a loss of the entire value, not just precision.
func handleFixedPointConversionError(err error) {
	switch err.(type) {
	case nil:
		return
	case fix.PositiveOverflowError:
		panic(&OverflowError{})
	case fix.NegativeOverflowError:
		panic(&UnderflowError{})
	case fix.UnderflowError:
		panic(&UnderflowError{})
	default:
		panic(err)
	}
}
