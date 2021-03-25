/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

// BlockType
//
var BlockType = &SimpleType{
	Name:                 "Block",
	QualifiedName:        "Block",
	TypeID:               "Block",
	IsInvalid:            false,
	IsResource:           false,
	Storable:             false,
	Equatable:            false,
	ExternallyReturnable: false,
	Members: func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			"height": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						UInt64Type,
						blockTypeHeightFieldDocString,
					)
				},
			},
			"view": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						UInt64Type,
						blockTypeViewFieldDocString,
					)
				},
			},
			"timestamp": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						UFix64Type,
						blockTypeTimestampFieldDocString,
					)
				},
			},
			"id": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						blockIDFieldType,
						blockTypeIdFieldDocString,
					)
				},
			},
		}
	},
}

const BlockIDSize = 32

var blockIDFieldType = &ConstantSizedType{
	Type: UInt8Type,
	Size: BlockIDSize,
}

const blockTypeHeightFieldDocString = `
The height of the block.

If the blockchain is viewed as a tree with the genesis block at the root, the height of a node is the number of edges between the node and the genesis block
`

const blockTypeViewFieldDocString = `
The view of the block.

It is a detail of the consensus algorithm. It is a monotonically increasing integer and counts rounds in the consensus algorithm. Since not all rounds result in a finalized block, the view number is strictly greater than or equal to the block height
`

const blockTypeTimestampFieldDocString = `
The ID of the block.

It is essentially the hash of the block
`

const blockTypeIdFieldDocString = `
The timestamp of the block.

It is the local clock time of the block proposer when it generates the block
`
