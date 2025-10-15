// Code generated from ccf.cdc. DO NOT EDIT.
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

const CCFTypeEncodeFunctionName = "encode"

var CCFTypeEncodeFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "input",
			TypeAnnotation: sema.NewTypeAnnotation(&sema.ReferenceType{
				Type:          sema.AnyType,
				Authorization: sema.UnauthorizedAccess,
			}),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: &sema.VariableSizedType{
				Type: sema.UInt8Type,
			},
		},
	),
}

const CCFTypeEncodeFunctionDocString = `
Encodes an encodable value to CCF.
Returns nil if the value cannot be encoded.
`

const CCFTypeName = "CCF"

var CCFType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         CCFTypeName,
		Kind:               common.CompositeKindContract,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			CCFType,
			sema.PrimitiveAccess(ast.AccessAll),
			CCFTypeEncodeFunctionName,
			CCFTypeEncodeFunctionType,
			CCFTypeEncodeFunctionDocString,
		),
	}

	CCFType.Members = sema.MembersAsMap(members)
	CCFType.Fields = sema.MembersFieldNames(members)
}
