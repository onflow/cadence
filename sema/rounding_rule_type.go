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

package sema

import "github.com/onflow/cadence/errors"

const RoundingRuleTypeName = "RoundingRule"

var RoundingRuleType = newNativeEnumType(
	RoundingRuleTypeName,
	UInt8Type,
	nil,
)

var RoundingRuleTypeAnnotation = NewTypeAnnotation(RoundingRuleType)

type RoundingRule uint8

// NOTE: only add new rules, do *NOT* change existing items,
// reuse raw values for other items, swap the order, etc.
//
// # Existing stored values use these raw values and should not change
//
// IMPORTANT: update RoundingRules
const (
	RoundingRuleTowardZero RoundingRule = iota
	RoundingRuleAwayFromZero
	RoundingRuleNearestHalfAway
	RoundingRuleNearestHalfEven

	// !!! *WARNING* !!!
	// ADD NEW RULES *BEFORE* THIS WARNING.
	// DO *NOT* ADD NEW RULES AFTER THIS LINE!
	RoundingRule_Count
)

var RoundingRules = []RoundingRule{
	RoundingRuleTowardZero,
	RoundingRuleAwayFromZero,
	RoundingRuleNearestHalfAway,
	RoundingRuleNearestHalfEven,
}

func (rule RoundingRule) Name() string {
	switch rule {
	case RoundingRuleTowardZero:
		return "towardZero"
	case RoundingRuleAwayFromZero:
		return "awayFromZero"
	case RoundingRuleNearestHalfAway:
		return "nearestHalfAway"
	case RoundingRuleNearestHalfEven:
		return "nearestHalfEven"
	}

	panic(errors.NewUnreachableError())
}

func (rule RoundingRule) RawValue() uint8 {
	switch rule {
	case RoundingRuleTowardZero:
		return 0
	case RoundingRuleAwayFromZero:
		return 1
	case RoundingRuleNearestHalfAway:
		return 2
	case RoundingRuleNearestHalfEven:
		return 3
	}

	panic(errors.NewUnreachableError())
}

func (rule RoundingRule) DocString() string {
	switch rule {
	case RoundingRuleTowardZero:
		return RoundingRuleTowardZeroDocString
	case RoundingRuleAwayFromZero:
		return RoundingRuleAwayFromZeroDocString
	case RoundingRuleNearestHalfAway:
		return RoundingRuleNearestHalfAwayDocString
	case RoundingRuleNearestHalfEven:
		return RoundingRuleNearestHalfEvenDocString
	}

	panic(errors.NewUnreachableError())
}

const RoundingRuleTowardZeroDocString = `
Round to the closest representable fixed-point value that has
a magnitude less than or equal to the magnitude of the real result,
effectively truncating the fractional part.

e.g. 5e-8 / 2 = 2e-8, -5e-8 / 2 = -2e-8
`

const RoundingRuleAwayFromZeroDocString = `
Round to the closest representable fixed-point value that has
a magnitude greater than or equal to the magnitude of the real result,
effectively rounding up any fractional part.

e.g. 5e-8 / 2 = 3e-8, -5e-8 / 2 = -3e-8
`

const RoundingRuleNearestHalfAwayDocString = `
Round to the closest representable fixed-point value to the real result,
which could be larger (rounded up) or smaller (rounded down) depending on
if the unrepresentable portion is greater than or less than one half
the difference between two available values.

If two representable values are equally close,
the value will be rounded away from zero.

e.g. 7e-8 / 2 = 4e-8, 5e-8 / 2 = 3e-8
`

const RoundingRuleNearestHalfEvenDocString = `
Round to the closest representable fixed-point value to the real result,
which could be larger (rounded up) or smaller (rounded down) depending on
if the unrepresentable portion is greater than or less than one half
the difference between two available values.

If two representable values are equally close,
the value with an even digit in the smallest decimal place will be chosen.

e.g. 7e-8 / 2 = 4e-8, 5e-8 / 2 = 2e-8
`
