// Code generated from rlp.cdc. DO NOT EDIT.
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

package stdlib

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

const RLPTypeDecodeStringFunctionName = "decodeString"

var RLPTypeDecodeStringFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "input",
			TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{
				Type: UInt8Type,
			}),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VariableSizedType{
			Type: UInt8Type,
		},
	),
}

const RLPTypeDecodeStringFunctionDocString = `
Decodes an RLP-encoded byte array (called string in the context of RLP).
The byte array should only contain of a single encoded value for a string;
if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
If any error is encountered while decoding, the program aborts.
`

const RLPTypeDecodeListFunctionName = "decodeList"

var RLPTypeDecodeListFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "input",
			TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{
				Type: UInt8Type,
			}),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VariableSizedType{
			Type: &sema.VariableSizedType{
				Type: UInt8Type,
			},
		},
	),
}

const RLPTypeDecodeListFunctionDocString = `
Decodes an RLP-encoded list into an array of RLP-encoded items.
Note that this function does not recursively decode, so each element of the resulting array is RLP-encoded data.
The byte array should only contain of a single encoded value for a list;
if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
If any error is encountered while decoding, the program aborts.
`

const RLPTypeName = "RLP"

var RLPType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         RLPTypeName,
		Kind:               common.CompositeKindContract,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			RLPType,
			ast.AccessPublic,
			RLPTypeDecodeStringFunctionName,
			RLPTypeDecodeStringFunctionType,
			RLPTypeDecodeStringFunctionDocString,
		),
		sema.NewUnmeteredFunctionMember(
			RLPType,
			ast.AccessPublic,
			RLPTypeDecodeListFunctionName,
			RLPTypeDecodeListFunctionType,
			RLPTypeDecodeListFunctionDocString,
		),
	}

	RLPType.Members = sema.MembersAsMap(members)
	RLPType.Fields = sema.MembersFieldNames(members)
}
