// Code generated from struct_stringer.cdc. DO NOT EDIT.
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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

const StructStringerTypeToStringFunctionName = "toString"

var StructStringerTypeToStringFunctionType = &FunctionType{
	Purity: FunctionPurityView,
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

const StructStringerTypeToStringFunctionDocString = `
Returns the string representation of this object.
`

const StructStringerTypeName = "StructStringer"

var StructStringerType = func() *InterfaceType {
	var t = &InterfaceType{
		Identifier:    StructStringerTypeName,
		CompositeKind: common.CompositeKindStructure,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			StructStringerType,
			PrimitiveAccess(ast.AccessAll),
			StructStringerTypeToStringFunctionName,
			StructStringerTypeToStringFunctionType,
			StructStringerTypeToStringFunctionDocString,
		),
	}

	StructStringerType.Members = MembersAsMap(members)
	StructStringerType.Fields = MembersFieldNames(members)
}
