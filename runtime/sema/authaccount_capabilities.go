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

var accountCapabilitiesTypeName = "Capabilities"

var AuthAccountCapabilitiesType = fix(func(authAccountCapabilitiesType *CompositeType) CompositeType {
	members := []*Member{
		NewUnmeteredPublicFunctionMember(
			authAccountCapabilitiesType,
			AccountCapabilitiesTypeGetFunctionName,
			AccountCapabilitiesTypeGetFunctionType,
			AccountCapabilitiesTypeGetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountCapabilitiesType,
			AccountCapabilitiesTypeBorrowFunctionName,
			AccountCapabilitiesTypeBorrowFunctionType,
			AccountCapabilitiesTypeBorrowFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountCapabilitiesType,
			AccountCapabilitiesTypeForEachFunctionName,
			AccountCapabilitiesTypeForEachFunctionType,
			AccountCapabilitiesTypeForEachFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountCapabilitiesType,
			AuthAccountCapabilitiesTypeGetControllerFunctionName,
			AuthAccountCapabilitiesTypeGetControllerFunctionType,
			AuthAccountCapabilitiesTypeGetControllerFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountCapabilitiesType,
			AuthAccountCapabilitiesTypeGetControllersFunctionName,
			AuthAccountCapabilitiesTypeGetControllersFunctionType,
			AuthAccountCapabilitiesTypeGetControllersFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountCapabilitiesType,
			AuthAccountCapabilitiesTypeForEachControllerFunctionName,
			AuthAccountCapabilitiesTypeForEachControllerFunctionType,
			AuthAccountCapabilitiesTypeForEachControllerFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountCapabilitiesType,
			AuthAccountCapabilitiesTypeIssueFunctionName,
			AuthAccountCapabilitiesTypeIssueFunctionType,
			AuthAccountCapabilitiesTypeIssueFunctionDocString,
		),
	}

	return CompositeType{
		Identifier: accountCapabilitiesTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
		Members:    GetMembersAsMap(members),
		Fields:     GetFieldNames(members),
	}
})

func init() {
	// Set the container type after initializing the `AuthAccountContractsType`, to avoid initializing loop.
	AuthAccountCapabilitiesType.SetContainerType(AuthAccountType)
}
