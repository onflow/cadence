/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package migrations

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// LegacyOptionalType simulates the old reference type with the old typeID generation.
type LegacyOptionalType struct {
	*interpreter.OptionalStaticType
}

var _ interpreter.StaticType = &LegacyOptionalType{}

func FormatOptionalTypeID[T ~string](elementTypeID T) T {
	return T(fmt.Sprintf("%s?", elementTypeID))
}

func (t *LegacyOptionalType) ID() common.TypeID {
	return FormatOptionalTypeID(t.Type.ID())
}

func (t *LegacyOptionalType) Encode(e *cbor.StreamEncoder) error {
	err := e.EncodeRawBytes([]byte{
		// tag number
		0xd8, interpreter.CBORTagOptionalStaticType,
	})
	if err != nil {
		return err
	}

	return t.Type.Encode(e)
}

func (t *LegacyOptionalType) Equal(other interpreter.StaticType) bool {
	if otherLegacy, ok := other.(*LegacyOptionalType); ok {
		other = otherLegacy.OptionalStaticType
	}
	return t.OptionalStaticType.Equal(other)
}
