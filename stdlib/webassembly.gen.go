// Code generated from webassembly.cdc. DO NOT EDIT.
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

package stdlib

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

const WebAssemblyTypeCompileAndInstantiateFunctionName = "compileAndInstantiate"

var WebAssemblyTypeCompileAndInstantiateFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []sema.Parameter{
		{
			Identifier: "bytes",
			TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{
				Type: sema.UInt8Type,
			}),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.ReferenceType{
			Type:          WebAssembly_InstantiatedSourceType,
			Authorization: sema.UnauthorizedAccess,
		},
	),
}

const WebAssemblyTypeCompileAndInstantiateFunctionDocString = `
Compile WebAssembly binary code into a Module and instantiate it.
Imports are not supported.
`

const WebAssembly_InstantiatedSourceTypeInstanceFieldName = "instance"

var WebAssembly_InstantiatedSourceTypeInstanceFieldType = &sema.ReferenceType{
	Type:          WebAssembly_InstanceType,
	Authorization: sema.UnauthorizedAccess,
}

const WebAssembly_InstantiatedSourceTypeInstanceFieldDocString = `
The instance.
`

const WebAssembly_InstantiatedSourceTypeName = "InstantiatedSource"

var WebAssembly_InstantiatedSourceType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         WebAssembly_InstantiatedSourceTypeName,
		Kind:               common.CompositeKindStructure,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFieldMember(
			WebAssembly_InstantiatedSourceType,
			sema.PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			WebAssembly_InstantiatedSourceTypeInstanceFieldName,
			WebAssembly_InstantiatedSourceTypeInstanceFieldType,
			WebAssembly_InstantiatedSourceTypeInstanceFieldDocString,
		),
	}

	WebAssembly_InstantiatedSourceType.Members = sema.MembersAsMap(members)
	WebAssembly_InstantiatedSourceType.Fields = sema.MembersFieldNames(members)
}

const WebAssembly_InstanceTypeGetExportFunctionName = "getExport"

var WebAssembly_InstanceTypeGetExportFunctionTypeParameterT = &sema.TypeParameter{
	Name:      "T",
	TypeBound: sema.AnyStructType,
}

var WebAssembly_InstanceTypeGetExportFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	TypeParameters: []*sema.TypeParameter{
		WebAssembly_InstanceTypeGetExportFunctionTypeParameterT,
	},
	Parameters: []sema.Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.GenericType{
			TypeParameter: WebAssembly_InstanceTypeGetExportFunctionTypeParameterT,
		},
	),
}

const WebAssembly_InstanceTypeGetExportFunctionDocString = `
Get the exported value.
The type must match the type of the exported value.
If the export with the given name does not exist,
of if the type does not match, then the function will panic.
`

const WebAssembly_InstanceTypeName = "Instance"

var WebAssembly_InstanceType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         WebAssembly_InstanceTypeName,
		Kind:               common.CompositeKindStructure,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			WebAssembly_InstanceType,
			sema.PrimitiveAccess(ast.AccessAll),
			WebAssembly_InstanceTypeGetExportFunctionName,
			WebAssembly_InstanceTypeGetExportFunctionType,
			WebAssembly_InstanceTypeGetExportFunctionDocString,
		),
	}

	WebAssembly_InstanceType.Members = sema.MembersAsMap(members)
	WebAssembly_InstanceType.Fields = sema.MembersFieldNames(members)
}

const WebAssemblyTypeName = "WebAssembly"

var WebAssemblyType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         WebAssemblyTypeName,
		Kind:               common.CompositeKindContract,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	t.SetNestedType(WebAssembly_InstantiatedSourceTypeName, WebAssembly_InstantiatedSourceType)
	t.SetNestedType(WebAssembly_InstanceTypeName, WebAssembly_InstanceType)
	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			WebAssemblyType,
			sema.PrimitiveAccess(ast.AccessAll),
			WebAssemblyTypeCompileAndInstantiateFunctionName,
			WebAssemblyTypeCompileAndInstantiateFunctionType,
			WebAssemblyTypeCompileAndInstantiateFunctionDocString,
		),
	}

	WebAssemblyType.Members = sema.MembersAsMap(members)
	WebAssemblyType.Fields = sema.MembersFieldNames(members)
}
