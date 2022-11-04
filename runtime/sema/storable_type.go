/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

// StorableType is the supertype of all types which are storable.
//
// It is only used as e.g. a type bound, but is not accessible
// to user programs, i.e. can't be used in type annotations
// for e.g. parameters, return types, fields, etc.
var StorableType = &SimpleType{
	Name:          "Storable",
	QualifiedName: "Storable",
	TypeID:        "Storable",
	IsInvalid:     false,
	// NOTE: Subtypes may be either resource types or not.
	//
	// Returning false here is safe, because this type is
	// only used as e.g. a type bound, but is not accessible
	// to user programs, i.e. can't be used in type annotations
	// for e.g. parameters, return types, fields, etc.
	IsResource:           false,
	Storable:             true,
	Equatable:            false,
	ExternallyReturnable: false,
	Importable:           false,
	IsSuperTypeOf: func(subType Type) bool {
		storableResults := map[*Member]bool{}
		return subType.IsStorable(storableResults)
	},
}
