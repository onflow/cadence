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

package bbq

import (
	"github.com/onflow/cadence/common"
)

// CanonicalName uniquely identifies a declaration by its location and its
// location-relative qualified name.
//
// Name is the declaration's simple name (e.g. "Foo" or "test").
//
// TypeQualifier is the name of the enclosing type, without location
// qualification (e.g. "Foo", "$ArrayVariableSized"). It is empty for
// top-level declarations.
//
// CanonicalName is used directly as a map key. Its identity therefore
// includes the complete location value, not only the TypeID rendered by
// String. In particular, address locations at the same address but with
// different program names remain distinct even when their rendered TypeIDs
// are identical.
type CanonicalName struct {
	Location      common.Location
	TypeQualifier string
	Name          string
}

func NewCanonicalName(location common.Location, name string) CanonicalName {
	return CanonicalName{
		Location: location,
		Name:     name,
	}
}

// NewTypedCanonicalName creates a CanonicalName for a member nested inside
// the given enclosing type. The type qualifier is the enclosing type's name
// without location qualification.
func NewTypedCanonicalName(
	location common.Location,
	typeQualifier string,
	name string,
) CanonicalName {
	return CanonicalName{
		Location:      location,
		TypeQualifier: typeQualifier,
		Name:          name,
	}
}

func (n CanonicalName) IsEmpty() bool {
	return n.Location == nil && n.Name == ""
}

func (n CanonicalName) TypeID(memoryGauge common.MemoryGauge) common.TypeID {
	name := n.Name
	if n.TypeQualifier != "" {
		name = n.TypeQualifier + "." + name
	}

	if n.Location == nil {
		return common.TypeID(name)
	}
	return n.Location.TypeID(memoryGauge, name)
}

func (n CanonicalName) String() string {
	return string(n.TypeID(nil))
}
