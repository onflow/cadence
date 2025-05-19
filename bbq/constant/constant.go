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

package constant

import (
	"bytes"
	"fmt"

	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type Constant struct {
	Data []byte
	Kind Kind
}

func (c Constant) String() string {
	// TODO: duplicate of `VM.initializeConstant()`
	kind := c.Kind

	// Prevent unintentional mutation of the original data
	data := bytes.Clone(c.Data)

	var value any

	switch kind {

	case String:
		value = interpreter.NewUnmeteredStringValue(string(data))

	case Character:
		value = interpreter.NewUnmeteredCharacterValue(string(data))

	case Int:
		value = interpreter.NewIntValueFromBigEndianBytes(nil, data)

	case Int8:
		value = interpreter.NewInt8ValueFromBigEndianBytes(nil, data)

	case Int16:
		value = interpreter.NewInt16ValueFromBigEndianBytes(nil, data)

	case Int32:
		value = interpreter.NewInt32ValueFromBigEndianBytes(nil, data)

	case Int64:
		value = interpreter.NewInt64ValueFromBigEndianBytes(nil, data)

	case Int128:
		value = interpreter.NewInt128ValueFromBigEndianBytes(nil, data)

	case Int256:
		value = interpreter.NewInt256ValueFromBigEndianBytes(nil, data)

	case UInt:
		value = interpreter.NewUIntValueFromBigEndianBytes(nil, data)

	case UInt8:
		value = interpreter.NewUInt8ValueFromBigEndianBytes(nil, data)

	case UInt16:
		value = interpreter.NewUInt16ValueFromBigEndianBytes(nil, data)

	case UInt32:
		value = interpreter.NewUInt32ValueFromBigEndianBytes(nil, data)

	case UInt64:
		value = interpreter.NewUInt64ValueFromBigEndianBytes(nil, data)

	case UInt128:
		value = interpreter.NewUInt128ValueFromBigEndianBytes(nil, data)

	case UInt256:
		value = interpreter.NewUInt256ValueFromBigEndianBytes(nil, data)

	case Word8:
		value = interpreter.NewWord8ValueFromBigEndianBytes(nil, data)

	case Word16:
		value = interpreter.NewWord16ValueFromBigEndianBytes(nil, data)

	case Word32:
		value = interpreter.NewWord32ValueFromBigEndianBytes(nil, data)

	case Word64:
		value = interpreter.NewWord64ValueFromBigEndianBytes(nil, data)

	case Word128:
		value = interpreter.NewWord128ValueFromBigEndianBytes(nil, data)

	case Word256:
		value = interpreter.NewWord256ValueFromBigEndianBytes(nil, data)

	case Fix64:
		value = interpreter.NewFix64ValueFromBigEndianBytes(nil, data)

	case UFix64:
		value = interpreter.NewUFix64ValueFromBigEndianBytes(nil, data)

	case Address:
		value = interpreter.NewUnmeteredAddressValueFromBytes(data)

	default:
		panic(errors.NewUnexpectedError("unsupported constant kind: %s", kind))
	}

	return fmt.Sprint(value)
}
