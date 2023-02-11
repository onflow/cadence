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

import "github.com/onflow/cadence/runtime/common"

// fixed-point recursion for building self-referential structures
func fix[T any](f func(*T) T) *T {
	var x T
	x = f(&x)
	return &x
}

const PublicAccountCapabilitiesTypeName = "Capabilities"

var PublicAccountCapabilitiesType = fix(func(publicAccountCapabilitiesType *CompositeType) CompositeType {
	members := []*Member{
		NewUnmeteredPublicFunctionMember(
			publicAccountCapabilitiesType,
			AccountCapabilitiesTypeGetFunctionName,
			AccountCapabilitiesTypeGetFunctionType,
			AccountCapabilitiesTypeGetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicAccountCapabilitiesType,
			AccountCapabilitiesTypeBorrowFunctionName,
			AccountCapabilitiesTypeBorrowFunctionType,
			AccountCapabilitiesTypeBorrowFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicAccountCapabilitiesType,
			AccountCapabilitiesTypeForEachFunctionName,
			AccountCapabilitiesTypeForEachFunctionType,
			AccountCapabilitiesTypeForEachFunctionDocString,
		),
	}

	return CompositeType{
		Identifier: PublicAccountCapabilitiesTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
		Members:    GetMembersAsMap(members),
		Fields:     GetFieldNames(members),
	}
})

func init() {
	// Set the container type after initializing the `PublicAccountContractsType`, to avoid initializing loop.
	PublicAccountCapabilitiesType.SetContainerType(PublicAccountType)
}
