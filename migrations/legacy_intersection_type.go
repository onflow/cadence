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
	"strings"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// LegacyIntersectionType simulates the old, incorrect restricted-type type-ID generation,
// which did not sort the type IDs of the interface types.
type LegacyIntersectionType struct {
	*interpreter.IntersectionStaticType
}

var _ interpreter.StaticType = &LegacyIntersectionType{}

func (t *LegacyIntersectionType) Equal(other interpreter.StaticType) bool {
	var otherTypes []*interpreter.InterfaceStaticType

	switch other := other.(type) {

	case *LegacyIntersectionType:
		otherTypes = other.Types

	case *interpreter.IntersectionStaticType:
		otherTypes = other.Types

	default:
		return false
	}

	if len(t.Types) != len(otherTypes) {
		return false
	}

outer:
	for _, typ := range t.Types {
		for _, otherType := range otherTypes {
			if typ.Equal(otherType) {
				continue outer
			}
		}

		return false
	}

	return true
}

func (t *LegacyIntersectionType) ID() common.TypeID {
	interfaceTypeIDs := make([]string, 0, len(t.Types))
	for _, interfaceType := range t.Types {
		interfaceTypeIDs = append(
			interfaceTypeIDs,
			string(interfaceType.ID()),
		)
	}

	var result strings.Builder

	if t.LegacyType != nil {
		result.WriteString(string(t.LegacyType.ID()))
	}

	result.WriteByte('{')
	// NOTE: no sorting
	for i, interfaceTypeID := range interfaceTypeIDs {
		if i > 0 {
			result.WriteByte(',')
		}
		result.WriteString(interfaceTypeID)
	}
	result.WriteByte('}')
	return common.TypeID(result.String())
}
