/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022-2023 Dapper Labs, Inc.
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

// CCF uses CBOR tag numbers 128-255, which are unassigned by [IANA]
// (https://www.iana.org/assignments/cbor-tags/cbor-tags.xhtml).

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
	CBORTagRestrictedType
	CBORTagCapabilityType
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
	_
	_
	_

	// composite types (165-175 are reserved)
	CBORTagStructType
	CBORTagResourceType
	CBORTagEventType
	CBORTagContractType
	CBORTagEnumType
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
	CBORTagRestrictedTypeValue
	CBORTagCapabilityTypeValue
	CBORTagFunctionTypeValue
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
	_
	_

	// composite type values (213-223 are reserved)
	CBORTagStructTypeValue
	CBORTagResourceTypeValue
	CBORTagEventTypeValue
	CBORTagContractTypeValue
	CBORTagEnumTypeValue
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
