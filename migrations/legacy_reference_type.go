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

package migrations

import (
	"strings"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// LegacyReferenceType simulates the old reference type with the old typeID generation.
type LegacyReferenceType struct {
	*interpreter.ReferenceStaticType
}

var _ interpreter.StaticType = &LegacyReferenceType{}

func (t *LegacyReferenceType) ID() common.TypeID {
	if !t.HasLegacyIsAuthorized {
		// Encode as a regular reference type
		return t.ReferenceStaticType.ID()
	}

	borrowedType := t.ReferencedType
	return common.TypeID(
		formatReferenceType(
			t.LegacyIsAuthorized,
			string(borrowedType.ID()),
		),
	)
}

func formatReferenceType(
	authorized bool,
	typeString string,
) string {
	var builder strings.Builder
	if authorized {
		builder.WriteString("auth")
	}
	builder.WriteByte('&')
	builder.WriteString(typeString)
	return builder.String()
}

func (t *LegacyReferenceType) Encode(e *cbor.StreamEncoder) error {
	if !t.HasLegacyIsAuthorized {
		// Encode as a regular reference type
		return t.ReferenceStaticType.Encode(e)
	}

	// Has legacy isAuthorized flag,
	// encode as a legacy reference type

	// Encode tag number and array head
	err := e.EncodeRawBytes([]byte{
		// tag number
		0xd8, interpreter.CBORTagReferenceStaticType,
		// array, 2 items follow
		0x82,
	})
	if err != nil {
		return err
	}

	// Encode the `LegacyIsAuthorized` flag instead of the `Authorization`.
	// This is how it was done in pre-1.0.
	// Decode already supports decoding this flag, for backward compatibility.
	// Encode authorized at array index encodedReferenceStaticTypeAuthorizedFieldKey
	err = e.EncodeBool(t.LegacyIsAuthorized)
	if err != nil {
		return err
	}

	// Encode type at array index encodedReferenceStaticTypeTypeFieldKey
	return t.ReferencedType.Encode(e)
}

func (t *LegacyReferenceType) Equal(other interpreter.StaticType) bool {
	if otherLegacy, ok := other.(*LegacyReferenceType); ok {
		other = otherLegacy.ReferenceStaticType
	}
	return t.ReferenceStaticType.Equal(other)
}
