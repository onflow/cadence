// Code generated from account_capability_controller.cdc. DO NOT EDIT.
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

import "github.com/onflow/cadence/runtime/ast"

const AccountCapabilityControllerTypeCapabilityFieldName = "capability"

var AccountCapabilityControllerTypeCapabilityFieldType = &CapabilityType{}

const AccountCapabilityControllerTypeCapabilityFieldDocString = `
The capability that is controlled by this controller.
`

const AccountCapabilityControllerTypeTagFieldName = "tag"

var AccountCapabilityControllerTypeTagFieldType = StringType

const AccountCapabilityControllerTypeTagFieldDocString = `
An arbitrary "tag" for the controller.
For example, it could be used to describe the purpose of the capability.
Empty by default.
`

const AccountCapabilityControllerTypeBorrowTypeFieldName = "borrowType"

var AccountCapabilityControllerTypeBorrowTypeFieldType = MetaType

const AccountCapabilityControllerTypeBorrowTypeFieldDocString = `
The type of the controlled capability, i.e. the T in ` + "`Capability<T>`" + `.
`

const AccountCapabilityControllerTypeCapabilityIDFieldName = "capabilityID"

var AccountCapabilityControllerTypeCapabilityIDFieldType = UInt64Type

const AccountCapabilityControllerTypeCapabilityIDFieldDocString = `
The identifier of the controlled capability.
All copies of a capability have the same ID.
`

const AccountCapabilityControllerTypeDeleteFunctionName = "delete"

var AccountCapabilityControllerTypeDeleteFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const AccountCapabilityControllerTypeDeleteFunctionDocString = `
Delete this capability controller,
and disable the controlled capability and its copies.

The controller will be deleted from storage,
but the controlled capability and its copies remain.

Once this function returns, the controller is no longer usable,
all further operations on the controller will panic.

Borrowing from the controlled capability or its copies will return nil.
`

const AccountCapabilityControllerTypeName = "AccountCapabilityController"

var AccountCapabilityControllerType = &SimpleType{
	Name:          AccountCapabilityControllerTypeName,
	QualifiedName: AccountCapabilityControllerTypeName,
	TypeID:        AccountCapabilityControllerTypeName,
	tag:           AccountCapabilityControllerTypeTag,
	IsResource:    false,
	Storable:      false,
	Equatable:     false,
	Comparable:    false,
	Exportable:    false,
	Importable:    false,
}

func init() {
	AccountCapabilityControllerType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredFieldMember(
				t,
				ast.AccessPublic,
				ast.VariableKindConstant,
				AccountCapabilityControllerTypeCapabilityFieldName,
				AccountCapabilityControllerTypeCapabilityFieldType,
				AccountCapabilityControllerTypeCapabilityFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessPublicSettable,
				ast.VariableKindVariable,
				AccountCapabilityControllerTypeTagFieldName,
				AccountCapabilityControllerTypeTagFieldType,
				AccountCapabilityControllerTypeTagFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessPublic,
				ast.VariableKindConstant,
				AccountCapabilityControllerTypeBorrowTypeFieldName,
				AccountCapabilityControllerTypeBorrowTypeFieldType,
				AccountCapabilityControllerTypeBorrowTypeFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessPublic,
				ast.VariableKindConstant,
				AccountCapabilityControllerTypeCapabilityIDFieldName,
				AccountCapabilityControllerTypeCapabilityIDFieldType,
				AccountCapabilityControllerTypeCapabilityIDFieldDocString,
			),
			NewUnmeteredFunctionMember(
				t,
				ast.AccessPublic,
				AccountCapabilityControllerTypeDeleteFunctionName,
				AccountCapabilityControllerTypeDeleteFunctionType,
				AccountCapabilityControllerTypeDeleteFunctionDocString,
			),
		})
	}
}
