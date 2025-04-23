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

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/format"
)

// BoolValue

type BoolValue bool

var _ Value = BoolValue(false)
var _ EquatableValue = BoolValue(false)
var _ ComparableValue[BoolValue] = BoolValue(false)
var _ atree.Value = BoolValue(false)
var _ atree.Storable = BoolValue(false)

const TrueValue = BoolValue(true)
const FalseValue = BoolValue(false)

func (BoolValue) isValue() {}

func (v BoolValue) String() string {
	return format.Bool(bool(v))
}

func (v BoolValue) Negate() BoolValue {
	if v == TrueValue {
		return FalseValue
	}
	return TrueValue
}

func (v BoolValue) Equal(other Value) bool {
	otherBool, ok := other.(BoolValue)
	if !ok {
		return false
	}
	return bool(v) == bool(otherBool)
}

func (v BoolValue) Less(other BoolValue) bool {
	return bool(!v && other)
}

func (v BoolValue) LessEqual(other BoolValue) bool {
	return bool(!v || other)
}

func (v BoolValue) Greater(other BoolValue) bool {
	return bool(v && !other)
}

func (v BoolValue) GreaterEqual(other BoolValue) bool {
	return bool(v || !other)
}

func (v BoolValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (v BoolValue) ByteSize() uint32 {
	return 1
}

func (v BoolValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (BoolValue) ChildStorables() []atree.Storable {
	return nil
}

// Encode encodes the value as a CBOR bool
func (v BoolValue) Encode(e *atree.Encoder) error {
	// NOTE: when updating, also update BoolValue.ByteSize
	return e.CBOR.EncodeBool(bool(v))
}
