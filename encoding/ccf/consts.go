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

package ccf

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common/bimap"
)

// CCF uses CBOR tag numbers 128-255, which are unassigned by [IANA]
// (https://www.iana.org/assignments/cbor-tags/cbor-tags.xhtml).
//
// !!! *WARNING* !!!
//
// CCF Codec *MUST* comply with CCF Specifications.  Relevant changes
// must be in sync between codec and specifications.
//
// Only add new tag number by:
// - replacing existing placeholders (`_`) with new tag number
//
// Only remove tag number by:
// - replace existing tag number with a placeholder `_`
//
// DO *NOT* REPLACE EXISTING TAG NUMBERS!
// DO *NOT* ADD NEW TAG NUMBERS IN BETWEEN!
// DO *NOT* APPEND NEW TAG NUMBERS AT END!
//
// By not appending tag numbers to the end, we have larger block of
// unused tag numbers if needed.  Tag numbers in 128-255 are
// unassigned in CBOR, and we currently use 128-231.  Since each
// group of tags in this range have reserved space available,
// there is no need to append new tag numbers in 232-255.

const (
	// CBOR tag numbers (128-135) for root objects (131-135 are reserved)
	CBORTagTypeDef = 128 + iota
	CBORTagTypeDefAndValue
	CBORTagTypeAndValue
	_
	_
	_
	_
	_

	// CBOR tag numbers (136-183) for types
	// inline types (145-159 are reserved)
	CBORTagTypeRef
	CBORTagSimpleType
	CBORTagOptionalType
	CBORTagVarsizedArrayType
	CBORTagConstsizedArrayType
	CBORTagDictType
	CBORTagReferenceType
	CBORTagIntersectionType
	CBORTagCapabilityType
	CBORTagInclusiveRangeType
	CBORTagEntitlementSetAuthorizationAccessType
	CBORTagEntitlementMapAuthorizationAccessType
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// composite types (165-175 are reserved)
	CBORTagStructType
	CBORTagResourceType
	CBORTagEventType
	CBORTagContractType
	CBORTagEnumType
	CBORTagAttachmentType
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// interface types (179-183 are reserved)
	CBORTagStructInterfaceType
	CBORTagResourceInterfaceType
	CBORTagContractInterfaceType
	_
	_
	_
	_
	_

	// CBOR tag numbers (184-231) for type value
	// non-composite and non-interface type values (194-207 are reserved)
	CBORTagTypeValueRef
	CBORTagSimpleTypeValue
	CBORTagOptionalTypeValue
	CBORTagVarsizedArrayTypeValue
	CBORTagConstsizedArrayTypeValue
	CBORTagDictTypeValue
	CBORTagReferenceTypeValue
	CBORTagIntersectionTypeValue
	CBORTagCapabilityTypeValue
	CBORTagFunctionTypeValue
	CBORTagInclusiveRangeTypeValue // InclusiveRange is stored as a composite value.
	CBORTagEntitlementSetAuthorizationAccessTypeValue
	CBORTagEntitlementMapAuthorizationAccessTypeValue
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// composite type values (213-223 are reserved)
	CBORTagStructTypeValue
	CBORTagResourceTypeValue
	CBORTagEventTypeValue
	CBORTagContractTypeValue
	CBORTagEnumTypeValue
	CBORTagAttachmentTypeValue
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// interface type values (227-231 are reserved)
	CBORTagStructInterfaceTypeValue
	CBORTagResourceInterfaceTypeValue
	CBORTagContractInterfaceTypeValue
	_
	_
	_
	_
	_
)

type entitlementSetKind uint64

const (
	conjunction entitlementSetKind = iota
	disjunction
)

func initEntitlementSetKindBiMap() (m *bimap.BiMap[cadence.EntitlementSetKind, entitlementSetKind]) {
	m = bimap.NewBiMap[cadence.EntitlementSetKind, entitlementSetKind]()

	m.Insert(cadence.Conjunction, conjunction)
	m.Insert(cadence.Disjunction, disjunction)

	return
}

var entitlementSetKindBiMap *bimap.BiMap[cadence.EntitlementSetKind, entitlementSetKind] = initEntitlementSetKindBiMap()

func entitlementSetKindRawValueByCadenceType(typ cadence.EntitlementSetKind) (entitlementSetKind, bool) {
	return entitlementSetKindBiMap.Get(typ)
}

func entitlementSetKindCadenceTypeByRawValue(rawValue entitlementSetKind) (cadence.EntitlementSetKind, bool) {
	return entitlementSetKindBiMap.GetInverse(rawValue)
}
