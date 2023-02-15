// Code generated from capability_controller.cdc. DO NOT EDIT.
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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

const CapabilityControllerTypeIssueHeightFieldName = "issueHeight"

var CapabilityControllerTypeIssueHeightFieldType = UInt64Type

const CapabilityControllerTypeIssueHeightFieldDocString = `The block height when the capability was created.
`

const CapabilityControllerTypeBorrowTypeFieldName = "borrowType"

var CapabilityControllerTypeBorrowTypeFieldType = MetaType

const CapabilityControllerTypeBorrowTypeFieldDocString = `The Type of the capability, i.e.: the T in Capability<T>.
`

const CapabilityControllerTypeCapabilityIDFieldName = "capabilityID"

var CapabilityControllerTypeCapabilityIDFieldType = UInt64Type

const CapabilityControllerTypeCapabilityIDFieldDocString = `The id of the related capability.
This is the UUID of the created capability.
All copies of the same capability will have the same UUID
`

const CapabilityControllerTypeIsRevokedFunctionName = "isRevoked"

var CapabilityControllerTypeIsRevokedFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		BoolType,
	),
}

const CapabilityControllerTypeIsRevokedFunctionDocString = `Is the capability revoked.
`

const CapabilityControllerTypeTargetFunctionName = "target"

var CapabilityControllerTypeTargetFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		StoragePathType,
	),
}

const CapabilityControllerTypeTargetFunctionDocString = `Returns the targeted storage path of the capability.
`

const CapabilityControllerTypeDeleteFunctionName = "delete"

var CapabilityControllerTypeDeleteFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const CapabilityControllerTypeDeleteFunctionDocString = `Delete this capability controller,
and disable the controlled capability and its copies.

The controller will be deleted from storage,
but the controlled capability and its copies remain.

Once this function returns, the controller is no longer usable,
all further operations on the controller will panic.

Borrowing from the controlled capability or its copies will return nil.
`

const CapabilityControllerTypeRetargetFunctionName = "retarget"

var CapabilityControllerTypeRetargetFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const CapabilityControllerTypeRetargetFunctionDocString = `Retarget the capability.
This moves the CapCon from one CapCon array to another.
`

const CapabilityControllerTypeName = "CapabilityController"

var CapabilityControllerType = &SimpleType{
	Name:          CapabilityControllerTypeName,
	QualifiedName: CapabilityControllerTypeName,
	TypeID:        CapabilityControllerTypeName,
	tag:           CapabilityControllerTypeTag,
	IsResource:    false,
	Storable:      false,
	Equatable:     false,
	Exportable:    false,
	Importable:    false,
	Members: func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			CapabilityControllerTypeIssueHeightFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						CapabilityControllerTypeIssueHeightFieldType,
						CapabilityControllerTypeIssueHeightFieldDocString,
					)
				},
			},
			CapabilityControllerTypeBorrowTypeFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						CapabilityControllerTypeBorrowTypeFieldType,
						CapabilityControllerTypeBorrowTypeFieldDocString,
					)
				},
			},
			CapabilityControllerTypeCapabilityIDFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						CapabilityControllerTypeCapabilityIDFieldType,
						CapabilityControllerTypeCapabilityIDFieldDocString,
					)
				},
			},
			CapabilityControllerTypeIsRevokedFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						CapabilityControllerTypeIsRevokedFunctionType,
						CapabilityControllerTypeIsRevokedFunctionDocString,
					)
				},
			},
			CapabilityControllerTypeTargetFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						CapabilityControllerTypeTargetFunctionType,
						CapabilityControllerTypeTargetFunctionDocString,
					)
				},
			},
			CapabilityControllerTypeDeleteFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						CapabilityControllerTypeDeleteFunctionType,
						CapabilityControllerTypeDeleteFunctionDocString,
					)
				},
			},
			CapabilityControllerTypeRetargetFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						CapabilityControllerTypeRetargetFunctionType,
						CapabilityControllerTypeRetargetFunctionDocString,
					)
				},
			},
		}
	},
}
