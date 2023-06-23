// Code generated from block.cdc. DO NOT EDIT.
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

const BlockTypeHeightFieldName = "height"

var BlockTypeHeightFieldType = UInt64Type

const BlockTypeHeightFieldDocString = `
The height of the block.

If the blockchain is viewed as a tree with the genesis block at the root,
the height of a node is the number of edges between the node and the genesis block
`

const BlockTypeViewFieldName = "view"

var BlockTypeViewFieldType = UInt64Type

const BlockTypeViewFieldDocString = `
The view of the block.

It is a detail of the consensus algorithm. It is a monotonically increasing integer and counts rounds in the consensus algorithm.
Since not all rounds result in a finalized block, the view number is strictly greater than or equal to the block height
`

const BlockTypeTimestampFieldName = "timestamp"

var BlockTypeTimestampFieldType = UFix64Type

const BlockTypeTimestampFieldDocString = `
Consider observing blocks' status changes off-chain yourself to get a more reliable value.
`

const BlockTypeIdFieldName = "id"

var BlockTypeIdFieldType = &ConstantSizedType{
	Type: UInt8Type,
	Size: 32,
}

const BlockTypeIdFieldDocString = `
The ID of the block.
It is essentially the hash of the block
`

const BlockTypeName = "Block"

var BlockType = &SimpleType{
	Name:             BlockTypeName,
	QualifiedName:    BlockTypeName,
	TypeID:           BlockTypeName,
	tag:              BlockTypeTag,
	IsResource:       false,
	Storable:         false,
	Equatable:        false,
	Comparable:       false,
	Exportable:       false,
	Importable:       false,
	MemberAccessible: true,
}

func init() {
	BlockType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredFieldMember(
				t,
				ast.AccessAll,
				ast.VariableKindConstant,
				BlockTypeHeightFieldName,
				BlockTypeHeightFieldType,
				BlockTypeHeightFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessAll,
				ast.VariableKindConstant,
				BlockTypeViewFieldName,
				BlockTypeViewFieldType,
				BlockTypeViewFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessAll,
				ast.VariableKindConstant,
				BlockTypeTimestampFieldName,
				BlockTypeTimestampFieldType,
				BlockTypeTimestampFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessAll,
				ast.VariableKindConstant,
				BlockTypeIdFieldName,
				BlockTypeIdFieldType,
				BlockTypeIdFieldDocString,
			),
		})
	}
}
