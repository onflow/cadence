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

package common

//go:generate go run golang.org/x/tools/cmd/stringer -type=MemoryKind -trimprefix=MemoryKind

// MemoryKind
type MemoryKind uint

const (
	MemoryKindUnknown MemoryKind = iota

	// Values

	MemoryKindAddressValue
	MemoryKindStringValue
	MemoryKindCharacterValue
	MemoryKindNumberValue
	MemoryKindArrayValueBase
	MemoryKindDictionaryValueBase
	MemoryKindCompositeValueBase
	MemoryKindSimpleCompositeValueBase
	MemoryKindOptionalValue
	MemoryKindTypeValue
	MemoryKindPathValue
	MemoryKindCapabilityValue
	MemoryKindStorageReferenceValue
	MemoryKindEphemeralReferenceValue
	MemoryKindInterpretedFunctionValue
	MemoryKindHostFunctionValue
	MemoryKindBoundFunctionValue
	MemoryKindBigInt
	MemoryKindSimpleCompositeValue
	MemoryKindPublishedValue
	MemoryKindStorageCapabilityControllerValue
	MemoryKindAccountCapabilityControllerValue

	// Atree Nodes
	MemoryKindAtreeArrayDataSlab
	MemoryKindAtreeArrayMetaDataSlab
	MemoryKindAtreeArrayElementOverhead
	MemoryKindAtreeMapDataSlab
	MemoryKindAtreeMapMetaDataSlab
	MemoryKindAtreeMapElementOverhead
	MemoryKindAtreeMapPreAllocatedElement
	MemoryKindAtreeEncodedSlab

	// Static Types
	MemoryKindPrimitiveStaticType
	MemoryKindCompositeStaticType
	MemoryKindInterfaceStaticType
	MemoryKindVariableSizedStaticType
	MemoryKindConstantSizedStaticType
	MemoryKindDictionaryStaticType
	MemoryKindInclusiveRangeStaticType
	MemoryKindOptionalStaticType
	MemoryKindIntersectionStaticType
	MemoryKindEntitlementSetStaticAccess
	MemoryKindEntitlementMapStaticAccess
	MemoryKindReferenceStaticType
	MemoryKindCapabilityStaticType
	MemoryKindFunctionStaticType

	// Cadence Values
	MemoryKindCadenceVoidValue
	MemoryKindCadenceOptionalValue
	MemoryKindCadenceBoolValue
	MemoryKindCadenceStringValue
	MemoryKindCadenceCharacterValue
	MemoryKindCadenceAddressValue
	MemoryKindCadenceIntValue
	MemoryKindCadenceNumberValue
	MemoryKindCadenceArrayValueBase
	MemoryKindCadenceArrayValueLength
	MemoryKindCadenceDictionaryValue
	MemoryKindCadenceInclusiveRangeValue
	MemoryKindCadenceKeyValuePair
	MemoryKindCadenceStructValueBase
	MemoryKindCadenceStructValueSize
	MemoryKindCadenceResourceValueBase
	MemoryKindCadenceAttachmentValueBase
	MemoryKindCadenceResourceValueSize
	MemoryKindCadenceAttachmentValueSize
	MemoryKindCadenceEventValueBase
	MemoryKindCadenceEventValueSize
	MemoryKindCadenceContractValueBase
	MemoryKindCadenceContractValueSize
	MemoryKindCadenceEnumValueBase
	MemoryKindCadenceEnumValueSize
	MemoryKindCadencePathValue
	MemoryKindCadenceTypeValue
	MemoryKindCadenceCapabilityValue
	MemoryKindCadenceFunctionValue

	// Cadence Types
	MemoryKindCadenceOptionalType
	MemoryKindCadenceVariableSizedArrayType
	MemoryKindCadenceConstantSizedArrayType
	MemoryKindCadenceDictionaryType
	MemoryKindCadenceInclusiveRangeType
	MemoryKindCadenceField
	MemoryKindCadenceParameter
	MemoryKindCadenceTypeParameter
	MemoryKindCadenceStructType
	MemoryKindCadenceResourceType
	MemoryKindCadenceAttachmentType
	MemoryKindCadenceEventType
	MemoryKindCadenceContractType
	MemoryKindCadenceStructInterfaceType
	MemoryKindCadenceResourceInterfaceType
	MemoryKindCadenceContractInterfaceType
	MemoryKindCadenceFunctionType
	MemoryKindCadenceEntitlementSetAccess
	MemoryKindCadenceEntitlementMapAccess
	MemoryKindCadenceReferenceType
	MemoryKindCadenceIntersectionType
	MemoryKindCadenceCapabilityType
	MemoryKindCadenceEnumType

	// Misc

	MemoryKindRawString
	MemoryKindAddressLocation
	MemoryKindBytes
	MemoryKindVariable
	MemoryKindCompositeTypeInfo
	MemoryKindCompositeField
	MemoryKindInvocation
	MemoryKindStorageMap
	MemoryKindStorageKey

	// Tokens

	MemoryKindTypeToken
	MemoryKindErrorToken
	MemoryKindSpaceToken

	// AST nodes

	MemoryKindProgram
	MemoryKindIdentifier
	MemoryKindArgument
	MemoryKindBlock
	MemoryKindFunctionBlock
	MemoryKindParameter
	MemoryKindParameterList
	MemoryKindTypeParameter
	MemoryKindTypeParameterList
	MemoryKindTransfer
	MemoryKindMembers
	MemoryKindTypeAnnotation
	MemoryKindDictionaryEntry

	MemoryKindFunctionDeclaration
	MemoryKindCompositeDeclaration
	MemoryKindAttachmentDeclaration
	MemoryKindInterfaceDeclaration
	MemoryKindEntitlementDeclaration
	MemoryKindEntitlementMappingElement
	MemoryKindEntitlementMappingDeclaration
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
	MemoryKindRemoveStatement

	MemoryKindBooleanExpression
	MemoryKindVoidExpression
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
	MemoryKindAttachExpression

	MemoryKindConstantSizedType
	MemoryKindDictionaryType
	MemoryKindFunctionType
	MemoryKindInstantiationType
	MemoryKindNominalType
	MemoryKindOptionalType
	MemoryKindReferenceType
	MemoryKindIntersectionType
	MemoryKindVariableSizedType

	MemoryKindPosition
	MemoryKindRange

	MemoryKindElaboration
	MemoryKindActivation
	MemoryKindActivationEntries

	// sema types
	MemoryKindVariableSizedSemaType
	MemoryKindConstantSizedSemaType
	MemoryKindDictionarySemaType
	MemoryKindOptionalSemaType
	MemoryKindIntersectionSemaType
	MemoryKindReferenceSemaType
	MemoryKindEntitlementSemaType
	MemoryKindEntitlementMapSemaType
	MemoryKindEntitlementRelationSemaType
	MemoryKindCapabilitySemaType
	MemoryKindInclusiveRangeSemaType

	// ordered-map
	MemoryKindOrderedMap
	MemoryKindOrderedMapEntryList
	MemoryKindOrderedMapEntry

	// Placeholder kind to allow consistent indexing
	// this should always be the last kind
	MemoryKindLast
)
