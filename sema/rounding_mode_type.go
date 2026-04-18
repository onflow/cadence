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

const RoundingModeTypeName = "RoundingMode"

var RoundingModeType = newNativeEnumType(
	RoundingModeTypeName,
	UInt8Type,
	nil,
)

var RoundingModeTypeAnnotation = NewTypeAnnotation(RoundingModeType)

type RoundingMode uint8

// NOTE: only add new modes, do *NOT* change existing items,
// reuse raw values for other items, swap the order, etc.
//
// # Existing stored values use these raw values and should not change
//
// IMPORTANT: update RoundingModes
const (
	RoundingModeTowardZero RoundingMode = iota
	RoundingModeAwayFromZero
	RoundingModeNearestHalfAway
	RoundingModeNearestHalfEven

	// !!! *WARNING* !!!
	// ADD NEW MODES *BEFORE* THIS WARNING.
	// DO *NOT* ADD NEW MODES AFTER THIS LINE!
	RoundingMode_Count
)

var RoundingModes = []RoundingMode{
	RoundingModeTowardZero,
	RoundingModeAwayFromZero,
	RoundingModeNearestHalfAway,
	RoundingModeNearestHalfEven,
}

func (mode RoundingMode) Name() string {
	switch mode {
	case RoundingModeTowardZero:
		return "towardZero"
	case RoundingModeAwayFromZero:
		return "awayFromZero"
	case RoundingModeNearestHalfAway:
		return "nearestHalfAway"
	case RoundingModeNearestHalfEven:
		return "nearestHalfEven"
	}

	panic(errors.NewUnreachableError())
}

func (mode RoundingMode) RawValue() uint8 {
	switch mode {
	case RoundingModeTowardZero:
		return 0
	case RoundingModeAwayFromZero:
		return 1
	case RoundingModeNearestHalfAway:
		return 2
	case RoundingModeNearestHalfEven:
		return 3
	}

	panic(errors.NewUnreachableError())
}

func (mode RoundingMode) DocString() string {
	switch mode {
	case RoundingModeTowardZero:
		return RoundingModeTowardZeroDocString
	case RoundingModeAwayFromZero:
		return RoundingModeAwayFromZeroDocString
	case RoundingModeNearestHalfAway:
		return RoundingModeNearestHalfAwayDocString
	case RoundingModeNearestHalfEven:
		return RoundingModeNearestHalfEvenDocString
	}

	panic(errors.NewUnreachableError())
}

const RoundingModeTowardZeroDocString = `
Round to the closest representable fixed-point value that has
a magnitude less than or equal to the magnitude of the real result,
effectively truncating the fractional part.

e.g. 5e-8 / 2 = 2e-8, -5e-8 / 2 = -2e-8
`

const RoundingModeAwayFromZeroDocString = `
Round to the closest representable fixed-point value that has
a magnitude greater than or equal to the magnitude of the real result,
effectively rounding up any fractional part.

e.g. 5e-8 / 2 = 3e-8, -5e-8 / 2 = -3e-8
`

const RoundingModeNearestHalfAwayDocString = `
Round to the closest representable fixed-point value to the real result,
which could be larger (rounded up) or smaller (rounded down) depending on
if the unrepresentable portion is greater than or less than one half
the difference between two available values.

If two representable values are equally close,
the value will be rounded away from zero.

e.g. 7e-8 / 2 = 4e-8, 5e-8 / 2 = 3e-8
`

const RoundingModeNearestHalfEvenDocString = `
Round to the closest representable fixed-point value to the real result,
which could be larger (rounded up) or smaller (rounded down) depending on
if the unrepresentable portion is greater than or less than one half
the difference between two available values.

If two representable values are equally close,
the value with an even digit in the smallest decimal place will be chosen.

e.g. 7e-8 / 2 = 4e-8, 5e-8 / 2 = 2e-8
`
