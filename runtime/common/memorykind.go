/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

package common

//go:generate go run golang.org/x/tools/cmd/stringer -type=MemoryKind -trimprefix=MemoryKind

// MemoryKind
//
type MemoryKind uint

const (
	MemoryKindUnknown MemoryKind = iota

	// Values

	MemoryKindBool
	MemoryKindAddress
	MemoryKindString
	MemoryKindCharacter
	MemoryKindMetaType
	MemoryKindNumber
	MemoryKindArrayBase
	MemoryKindArrayLength
	MemoryKindDictionaryBase
	MemoryKindDictionarySize
	MemoryKindCompositeBase
	MemoryKindCompositeSize
	MemoryKindOptional
	MemoryKindNil
	MemoryKindVoid
	MemoryKindTypeValue
	MemoryKindPathValue
	MemoryKindCapabilityValue
	MemoryKindLinkValue
	MemoryKindStorageReferenceValue
	MemoryKindEphemeralReferenceValue
	MemoryKindInterpretedFunction
	MemoryKindHostFunction
	MemoryKindBoundFunction
	MemoryKindBigInt

	// Static Types
	MemoryKindPrimitiveStaticType
	MemoryKindCompositeStaticType
	MemoryKindInterfaceStaticType
	MemoryKindVariableSizedStaticType
	MemoryKindConstantSizedStaticType
	MemoryKindDictionaryStaticType
	MemoryKindOptionalStaticType
	MemoryKindRestrictedStaticType
	MemoryKindReferenceStaticType
	MemoryKindCapabilityStaticType
	MemoryKindFunctionStaticType

	// Cadence Values
	MemoryKindCadenceVoid
	MemoryKindCadenceOptional
	MemoryKindCadenceBool
	MemoryKindCadenceString
	MemoryKindCadenceCharacter
	MemoryKindCadenceAddress
	MemoryKindCadenceInt
	MemoryKindCadenceNumber
	MemoryKindCadenceArrayBase
	MemoryKindCadenceArrayLength
	MemoryKindCadenceDictionaryBase
	MemoryKindCadenceDictionarySize

	// Misc

	MemoryKindRawString
	MemoryKindAddressLocation
	MemoryKindBytes
	MemoryKindVariable

	// Tokens

	MemoryKindValueToken
	MemoryKindSyntaxToken
	MemoryKindSpaceToken

	// AST nodes

	MemoryKindProgram
	MemoryKindIdentifier
	MemoryKindArgument
	MemoryKindBlock
	MemoryKindFunctionBlock
	MemoryKindParameter
	MemoryKindParameterList
	MemoryKindTransfer
	MemoryKindMembers
	MemoryKindTypeAnnotation
	MemoryKindDictionaryEntry

	MemoryKindFunctionDeclaration
	MemoryKindCompositeDeclaration
	MemoryKindInterfaceDeclaration
	MemoryKindEnumCaseDeclaration
	MemoryKindFieldDeclaration
	MemoryKindTransactionDeclaration
	MemoryKindImportDeclaration
	MemoryKindVariableDeclaration
	MemoryKindSpecialFunctionDeclaration
	MemoryKindPragmaDeclaration

	MemoryKindAssignmentStatement
	MemoryKindBreakStatement
	MemoryKindContinueStatement
	MemoryKindEmitStatement
	MemoryKindExpressionStatement
	MemoryKindForStatement
	MemoryKindIfStatement
	MemoryKindReturnStatement
	MemoryKindSwapStatement
	MemoryKindSwitchStatement
	MemoryKindWhileStatement

	MemoryKindBooleanExpression
	MemoryKindNilExpression
	MemoryKindStringExpression
	MemoryKindIntegerExpression
	MemoryKindFixedPointExpression
	MemoryKindArrayExpression
	MemoryKindDictionaryExpression
	MemoryKindIdentifierExpression
	MemoryKindInvocationExpression
	MemoryKindMemberExpression
	MemoryKindIndexExpression
	MemoryKindConditionalExpression
	MemoryKindUnaryExpression
	MemoryKindBinaryExpression
	MemoryKindFunctionExpression
	MemoryKindCastingExpression
	MemoryKindCreateExpression
	MemoryKindDestroyExpression
	MemoryKindReferenceExpression
	MemoryKindForceExpression
	MemoryKindPathExpression

	MemoryKindConstantSizedType
	MemoryKindDictionaryType
	MemoryKindFunctionType
	MemoryKindInstantiationType
	MemoryKindNominalType
	MemoryKindOptionalType
	MemoryKindReferenceType
	MemoryKindRestrictedType
	MemoryKindVariableSizedType

	MemoryKindPosition
	MemoryKindRange

	MemoryKindElaboration

	// Placeholder kind to allow consistent indexing
	// this should always be the last kind
	MemoryKindLast
)
