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
	"fmt"

	"github.com/onflow/cadence/bbq/leb128"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

type Constant struct {
	Data []byte
	Kind Kind
}

func (c Constant) String() string {
	// TODO: duplicate of `VM.initializeConstant()`
	kind := c.Kind
	data := c.Data

	var (
		v   any
		err error
	)

	switch kind {
	case String:
		v = string(data)

	case Character:
		v = string(data)

	case Int:
		// TODO: support larger integers
		v, _, err = leb128.ReadInt64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int constant: %s", err))
		}

	case Int8:
		v, _, err = leb128.ReadInt32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int8 constant: %s", err))
		}

	case Int16:
		v, _, err = leb128.ReadInt32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int16 constant: %s", err))
		}

	case Int32:
		v, _, err = leb128.ReadInt32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int32 constant: %s", err))
		}

	case Int64:
		v, _, err = leb128.ReadInt64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int64 constant: %s", err))
		}

	case UInt:
		// TODO: support larger integers
		v, _, err = leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt constant: %s", err))
		}

	case UInt8:
		v, _, err = leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt8 constant: %s", err))
		}

	case UInt16:
		v, _, err = leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt16 constant: %s", err))
		}

	case UInt32:
		v, _, err = leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt32 constant: %s", err))
		}

	case UInt64:
		v, _, err = leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt64 constant: %s", err))
		}

	case Word8:
		v, _, err = leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word8 constant: %s", err))
		}

	case Word16:
		v, _, err = leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word16 constant: %s", err))
		}

	case Word32:
		v, _, err = leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word32 constant: %s", err))
		}

	case Word64:
		v, _, err = leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word64 constant: %s", err))
		}

	case Fix64:
		v, _, err = leb128.ReadInt64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Fix64 constant: %s", err))
		}

	case UFix64:
		v, _, err = leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UFix64 constant: %s", err))
		}

	case Address:
		v = common.MustBytesToAddress(data)

	// TODO:
	// case constantkind.Int128:
	// case constantkind.Int256:
	// case constantkind.UInt128:
	// case constantkind.UInt256:
	// case constantkind.Word128:
	// case constantkind.Word256:

	default:
		panic(errors.NewUnexpectedError("unsupported constant kind: %s", kind))
	}

	return fmt.Sprint(v)
}
