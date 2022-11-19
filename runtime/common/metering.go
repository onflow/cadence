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

import (
	"math"
	"math/big"
	"unsafe"

	"github.com/onflow/cadence/runtime/errors"
)

type MemoryUsage struct {
	Kind   MemoryKind
	Amount uint64
}

type MemoryGauge interface {
	MeterMemory(usage MemoryUsage) error
}

var (
	// Tokens

	TypeTokenMemoryUsage  = NewConstantMemoryUsage(MemoryKindTypeToken)
	ErrorTokenMemoryUsage = NewConstantMemoryUsage(MemoryKindErrorToken)
	SpaceTokenMemoryUsage = NewConstantMemoryUsage(MemoryKindSpaceToken)

	// AST

	ProgramMemoryUsage         = NewConstantMemoryUsage(MemoryKindProgram)
	IdentifierMemoryUsage      = NewConstantMemoryUsage(MemoryKindIdentifier)
	ArgumentMemoryUsage        = NewConstantMemoryUsage(MemoryKindArgument)
	BlockMemoryUsage           = NewConstantMemoryUsage(MemoryKindBlock)
	FunctionBlockMemoryUsage   = NewConstantMemoryUsage(MemoryKindFunctionBlock)
	ParameterMemoryUsage       = NewConstantMemoryUsage(MemoryKindParameter)
	ParameterListMemoryUsage   = NewConstantMemoryUsage(MemoryKindParameterList)
	TransferMemoryUsage        = NewConstantMemoryUsage(MemoryKindTransfer)
	TypeAnnotationMemoryUsage  = NewConstantMemoryUsage(MemoryKindTypeAnnotation)
	DictionaryEntryMemoryUsage = NewConstantMemoryUsage(MemoryKindDictionaryEntry)

	// AST Declarations

	FunctionDeclarationMemoryUsage        = NewConstantMemoryUsage(MemoryKindFunctionDeclaration)
	CompositeDeclarationMemoryUsage       = NewConstantMemoryUsage(MemoryKindCompositeDeclaration)
	InterfaceDeclarationMemoryUsage       = NewConstantMemoryUsage(MemoryKindInterfaceDeclaration)
	ImportDeclarationMemoryUsage          = NewConstantMemoryUsage(MemoryKindImportDeclaration)
	TransactionDeclarationMemoryUsage     = NewConstantMemoryUsage(MemoryKindTransactionDeclaration)
	FieldDeclarationMemoryUsage           = NewConstantMemoryUsage(MemoryKindFieldDeclaration)
	EnumCaseDeclarationMemoryUsage        = NewConstantMemoryUsage(MemoryKindEnumCaseDeclaration)
	VariableDeclarationMemoryUsage        = NewConstantMemoryUsage(MemoryKindVariableDeclaration)
	SpecialFunctionDeclarationMemoryUsage = NewConstantMemoryUsage(MemoryKindSpecialFunctionDeclaration)
	PragmaDeclarationMemoryUsage          = NewConstantMemoryUsage(MemoryKindPragmaDeclaration)

	// AST Statements

	AssignmentStatementMemoryUsage = NewConstantMemoryUsage(MemoryKindAssignmentStatement)
	BreakStatementMemoryUsage      = NewConstantMemoryUsage(MemoryKindBreakStatement)
	ContinueStatementMemoryUsage   = NewConstantMemoryUsage(MemoryKindContinueStatement)
	EmitStatementMemoryUsage       = NewConstantMemoryUsage(MemoryKindEmitStatement)
	ExpressionStatementMemoryUsage = NewConstantMemoryUsage(MemoryKindExpressionStatement)
	ForStatementMemoryUsage        = NewConstantMemoryUsage(MemoryKindForStatement)
	IfStatementMemoryUsage         = NewConstantMemoryUsage(MemoryKindIfStatement)
	ReturnStatementMemoryUsage     = NewConstantMemoryUsage(MemoryKindReturnStatement)
	SwapStatementMemoryUsage       = NewConstantMemoryUsage(MemoryKindSwapStatement)
	SwitchStatementMemoryUsage     = NewConstantMemoryUsage(MemoryKindSwitchStatement)
	WhileStatementMemoryUsage      = NewConstantMemoryUsage(MemoryKindWhileStatement)

	// AST Expressions

	BooleanExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindBooleanExpression)
	NilExpressionMemoryUsage         = NewConstantMemoryUsage(MemoryKindNilExpression)
	StringExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindStringExpression)
	IntegerExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindIntegerExpression)
	FixedPointExpressionMemoryUsage  = NewConstantMemoryUsage(MemoryKindFixedPointExpression)
	IdentifierExpressionMemoryUsage  = NewConstantMemoryUsage(MemoryKindIdentifierExpression)
	InvocationExpressionMemoryUsage  = NewConstantMemoryUsage(MemoryKindInvocationExpression)
	MemberExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindMemberExpression)
	IndexExpressionMemoryUsage       = NewConstantMemoryUsage(MemoryKindIndexExpression)
	ConditionalExpressionMemoryUsage = NewConstantMemoryUsage(MemoryKindConditionalExpression)
	UnaryExpressionMemoryUsage       = NewConstantMemoryUsage(MemoryKindUnaryExpression)
	BinaryExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindBinaryExpression)
	FunctionExpressionMemoryUsage    = NewConstantMemoryUsage(MemoryKindFunctionExpression)
	CastingExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindCastingExpression)
	CreateExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindCreateExpression)
	DestroyExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindDestroyExpression)
	ReferenceExpressionMemoryUsage   = NewConstantMemoryUsage(MemoryKindReferenceExpression)
	ForceExpressionMemoryUsage       = NewConstantMemoryUsage(MemoryKindForceExpression)
	PathExpressionMemoryUsage        = NewConstantMemoryUsage(MemoryKindPathExpression)

	// AST Types

	ConstantSizedTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindConstantSizedType)
	DictionaryTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindDictionaryType)
	FunctionTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindFunctionType)
	InstantiationTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindInstantiationType)
	NominalTypeMemoryUsage       = NewConstantMemoryUsage(MemoryKindNominalType)
	OptionalTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindOptionalType)
	ReferenceTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindReferenceType)
	RestrictedTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindRestrictedType)
	VariableSizedTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindVariableSizedType)

	PositionMemoryUsage = NewConstantMemoryUsage(MemoryKindPosition)
	RangeMemoryUsage    = NewConstantMemoryUsage(MemoryKindRange)

	ElaborationMemoryUsage       = NewConstantMemoryUsage(MemoryKindElaboration)
	ActivationMemoryUsage        = NewConstantMemoryUsage(MemoryKindActivation)
	ActivationEntriesMemoryUsage = NewConstantMemoryUsage(MemoryKindActivationEntries)

	// Interpreter values

	SimpleCompositeValueBaseMemoryUsage = NewConstantMemoryUsage(MemoryKindSimpleCompositeValueBase)
	AtreeMapElementOverhead             = NewConstantMemoryUsage(MemoryKindAtreeMapElementOverhead)
	AtreeArrayElementOverhead           = NewConstantMemoryUsage(MemoryKindAtreeArrayElementOverhead)
	CompositeTypeInfoMemoryUsage        = NewConstantMemoryUsage(MemoryKindCompositeTypeInfo)
	CompositeFieldMemoryUsage           = NewConstantMemoryUsage(MemoryKindCompositeField)
	DictionaryValueBaseMemoryUsage      = NewConstantMemoryUsage(MemoryKindDictionaryValueBase)
	ArrayValueBaseMemoryUsage           = NewConstantMemoryUsage(MemoryKindArrayValueBase)
	CompositeValueBaseMemoryUsage       = NewConstantMemoryUsage(MemoryKindCompositeValueBase)
	AddressValueMemoryUsage             = NewConstantMemoryUsage(MemoryKindAddressValue)
	BoolValueMemoryUsage                = NewConstantMemoryUsage(MemoryKindBoolValue)
	NilValueMemoryUsage                 = NewConstantMemoryUsage(MemoryKindNilValue)
	VoidValueMemoryUsage                = NewConstantMemoryUsage(MemoryKindVoidValue)
	BoundFunctionValueMemoryUsage       = NewConstantMemoryUsage(MemoryKindBoundFunctionValue)
	HostFunctionValueMemoryUsage        = NewConstantMemoryUsage(MemoryKindHostFunctionValue)
	InterpretedFunctionValueMemoryUsage = NewConstantMemoryUsage(MemoryKindInterpretedFunctionValue)
	StorageCapabilityValueMemoryUsage   = NewConstantMemoryUsage(MemoryKindStorageCapabilityValue)
	EphemeralReferenceValueMemoryUsage  = NewConstantMemoryUsage(MemoryKindEphemeralReferenceValue)
	StorageReferenceValueMemoryUsage    = NewConstantMemoryUsage(MemoryKindStorageReferenceValue)
	LinkValueMemoryUsage                = NewConstantMemoryUsage(MemoryKindLinkValue)
	AccountLinkValueMemoryUsage         = NewConstantMemoryUsage(MemoryKindAccountLinkValue)
	PathValueMemoryUsage                = NewConstantMemoryUsage(MemoryKindPathValue)
	OptionalValueMemoryUsage            = NewConstantMemoryUsage(MemoryKindOptionalValue)
	TypeValueMemoryUsage                = NewConstantMemoryUsage(MemoryKindTypeValue)
	PublishedValueMemoryUsage           = NewConstantMemoryUsage(MemoryKindPublishedValue)

	// Static Types

	PrimitiveStaticTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindPrimitiveStaticType)
	CompositeStaticTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindCompositeStaticType)
	InterfaceStaticTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindInterfaceStaticType)
	VariableSizedStaticTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindVariableSizedStaticType)
	ConstantSizedStaticTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindConstantSizedStaticType)
	DictionaryStaticTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindDictionaryStaticType)
	OptionalStaticTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindOptionalStaticType)
	RestrictedStaticTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindRestrictedStaticType)
	ReferenceStaticTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindReferenceStaticType)
	CapabilityStaticTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindCapabilityStaticType)
	FunctionStaticTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindFunctionStaticType)

	// Sema types

	VariableSizedSemaTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindVariableSizedSemaType)
	ConstantSizedSemaTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindConstantSizedSemaType)
	DictionarySemaTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindDictionarySemaType)
	OptionalSemaTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindOptionalSemaType)
	RestrictedSemaTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindRestrictedSemaType)
	ReferenceSemaTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindReferenceSemaType)
	CapabilitySemaTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindCapabilitySemaType)

	// Storage related memory usages

	OrderedMapMemoryUsage = NewConstantMemoryUsage(MemoryKindOrderedMap)
	InvocationMemoryUsage = NewConstantMemoryUsage(MemoryKindInvocation)
	StorageMapMemoryUsage = NewConstantMemoryUsage(MemoryKindStorageMap)
	StorageKeyMemoryUsage = NewConstantMemoryUsage(MemoryKindStorageKey)

	// Cadence external values

	CadenceDictionaryValueMemoryUsage        = NewConstantMemoryUsage(MemoryKindCadenceDictionaryValue)
	CadenceArrayValueBaseMemoryUsage         = NewConstantMemoryUsage(MemoryKindCadenceArrayValueBase)
	CadenceStructValueBaseMemoryUsage        = NewConstantMemoryUsage(MemoryKindCadenceStructValueBase)
	CadenceResourceValueBaseMemoryUsage      = NewConstantMemoryUsage(MemoryKindCadenceResourceValueBase)
	CadenceEventValueBaseMemoryUsage         = NewConstantMemoryUsage(MemoryKindCadenceEventValueBase)
	CadenceContractValueBaseMemoryUsage      = NewConstantMemoryUsage(MemoryKindCadenceContractValueBase)
	CadenceEnumValueBaseMemoryUsage          = NewConstantMemoryUsage(MemoryKindCadenceEnumValueBase)
	CadenceAddressValueMemoryUsage           = NewConstantMemoryUsage(MemoryKindCadenceAddressValue)
	CadenceBoolValueMemoryUsage              = NewConstantMemoryUsage(MemoryKindCadenceBoolValue)
	CadenceStorageCapabilityValueMemoryUsage = NewConstantMemoryUsage(MemoryKindCadenceStorageCapabilityValue)
	CadenceFunctionValueMemoryUsage          = NewConstantMemoryUsage(MemoryKindCadenceFunctionValue)
	CadenceKeyValuePairMemoryUsage           = NewConstantMemoryUsage(MemoryKindCadenceKeyValuePair)
	CadenceLinkValueMemoryUsage              = NewConstantMemoryUsage(MemoryKindCadenceLinkValue)
	CadenceOptionalValueMemoryUsage          = NewConstantMemoryUsage(MemoryKindCadenceOptionalValue)
	CadencePathValueMemoryUsage              = NewConstantMemoryUsage(MemoryKindCadencePathValue)
	CadenceVoidValueMemoryUsage              = NewConstantMemoryUsage(MemoryKindCadenceVoidValue)
	CadenceTypeValueMemoryUsage              = NewConstantMemoryUsage(MemoryKindCadenceTypeValue)

	// Cadence external types

	CadenceSimpleTypeMemoryUsage             = NewConstantMemoryUsage(MemoryKindCadenceSimpleType)
	CadenceCapabilityTypeMemoryUsage         = NewConstantMemoryUsage(MemoryKindCadenceCapabilityType)
	CadenceConstantSizedArrayTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindCadenceConstantSizedArrayType)
	CadenceVariableSizedArrayTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindCadenceVariableSizedArrayType)
	CadenceContractInterfaceTypeMemoryUsage  = NewConstantMemoryUsage(MemoryKindCadenceContractInterfaceType)
	CadenceContractTypeMemoryUsage           = NewConstantMemoryUsage(MemoryKindCadenceContractType)
	CadenceDictionaryTypeMemoryUsage         = NewConstantMemoryUsage(MemoryKindCadenceDictionaryType)
	CadenceEnumTypeMemoryUsage               = NewConstantMemoryUsage(MemoryKindCadenceEnumType)
	CadenceEventTypeMemoryUsage              = NewConstantMemoryUsage(MemoryKindCadenceEventType)
	CadenceFunctionTypeMemoryUsage           = NewConstantMemoryUsage(MemoryKindCadenceFunctionType)
	CadenceOptionalTypeMemoryUsage           = NewConstantMemoryUsage(MemoryKindCadenceOptionalType)
	CadenceReferenceTypeMemoryUsage          = NewConstantMemoryUsage(MemoryKindCadenceReferenceType)
	CadenceResourceInterfaceTypeMemoryUsage  = NewConstantMemoryUsage(MemoryKindCadenceResourceInterfaceType)
	CadenceResourceTypeMemoryUsage           = NewConstantMemoryUsage(MemoryKindCadenceResourceType)
	CadenceRestrictedTypeMemoryUsage         = NewConstantMemoryUsage(MemoryKindCadenceRestrictedType)
	CadenceStructInterfaceTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindCadenceStructInterfaceType)
	CadenceStructTypeMemoryUsage             = NewConstantMemoryUsage(MemoryKindCadenceStructType)

	// Following are the known memory usage amounts for string representation of interpreter values.
	// Same as `len(format.X)`. However, values are hard-coded to avoid the circular dependency.

	VoidStringMemoryUsage                   = NewRawStringMemoryUsage(len("()"))
	TrueStringMemoryUsage                   = NewRawStringMemoryUsage(len("true"))
	FalseStringMemoryUsage                  = NewRawStringMemoryUsage(len("false"))
	TypeValueStringMemoryUsage              = NewRawStringMemoryUsage(len("Type<>()"))
	NilValueStringMemoryUsage               = NewRawStringMemoryUsage(len("nil"))
	StorageReferenceValueStringMemoryUsage  = NewRawStringMemoryUsage(len("StorageReference()"))
	SeenReferenceStringMemoryUsage          = NewRawStringMemoryUsage(3)                   // len(ellipsis)
	AddressValueStringMemoryUsage           = NewRawStringMemoryUsage(AddressLength*2 + 2) // len(bytes-to-hex + prefix)
	HostFunctionValueStringMemoryUsage      = NewRawStringMemoryUsage(len("Function(...)"))
	AuthAccountValueStringMemoryUsage       = NewRawStringMemoryUsage(len("AuthAccount()"))
	PublicAccountValueStringMemoryUsage     = NewRawStringMemoryUsage(len("PublicAccount()"))
	AuthAccountContractsStringMemoryUsage   = NewRawStringMemoryUsage(len("AuthAccount.Contracts()"))
	PublicAccountContractsStringMemoryUsage = NewRawStringMemoryUsage(len("PublicAccount.Contracts()"))
	AuthAccountKeysStringMemoryUsage        = NewRawStringMemoryUsage(len("AuthAccount.Keys()"))
	PublicAccountKeysStringMemoryUsage      = NewRawStringMemoryUsage(len("PublicAccount.Keys()"))
	AuthAccountInboxStringMemoryUsage       = NewRawStringMemoryUsage(len("AuthAccount.Inbox()"))
	StorageCapabilityValueStringMemoryUsage = NewRawStringMemoryUsage(len("Capability<>(address: , path: )"))
	LinkValueStringMemoryUsage              = NewRawStringMemoryUsage(len("Link<>()"))
	PublishedValueStringMemoryUsage         = NewRawStringMemoryUsage(len("PublishedValue<>()"))

	// Static types string representations

	VariableSizedStaticTypeStringMemoryUsage = NewRawStringMemoryUsage(2)  // []
	DictionaryStaticTypeStringMemoryUsage    = NewRawStringMemoryUsage(4)  // {: }
	OptionalStaticTypeStringMemoryUsage      = NewRawStringMemoryUsage(1)  // ?
	AuthReferenceStaticTypeStringMemoryUsage = NewRawStringMemoryUsage(5)  // auth&
	ReferenceStaticTypeStringMemoryUsage     = NewRawStringMemoryUsage(1)  // &
	CapabilityStaticTypeStringMemoryUsage    = NewRawStringMemoryUsage(12) // Capability<>
)

func UseMemory(gauge MemoryGauge, usage MemoryUsage) {
	if gauge == nil {
		return
	}

	err := gauge.MeterMemory(usage)
	if err != nil {
		panic(errors.MemoryError{Err: err})
	}
}

func NewConstantMemoryUsage(kind MemoryKind) MemoryUsage {
	return MemoryUsage{
		Kind:   kind,
		Amount: 1,
	}
}

func atreeNodes(count uint64, elementSize uint) (leafNodeCount uint64, branchNodeCount uint64) {
	if elementSize != 0 {
		// If we know how large each element is, we can compute the number of
		// atree leaf nodes using this formula:
		// size * elementSize / default_slab_size
		leafNodeCount = uint64(math.Ceil(float64(count) * float64(elementSize) / 1024))
	} else {
		// If we don't know how large each element is, we can overestimate
		// the number of atree leaf nodes this way, since every non-root leaf node
		// will always contain at least two elements.
		leafNodeCount = uint64(math.Ceil(float64(count) / 2))
	}
	if leafNodeCount < 1 {
		leafNodeCount = 1 // there will always be at least one data slab
	}

	branchNodeCount = 0
	if leafNodeCount >= 2 {
		const n = 20 // n-ary tree
		// Compute atree height from number of leaf nodes and n-ary
		height := 1 + int(math.Ceil(math.Log(float64(leafNodeCount))/math.Log(float64(n))))

		// Compute number of branch nodes from leaf node count and height
		for i := 1; i < height; i++ {
			branchNodeCount += uint64(math.Ceil(float64(leafNodeCount) / math.Pow(float64(n), float64(i))))
		}
	}
	return
}

func newAtreeMemoryUsage(count uint64, elementSize uint, array bool) (MemoryUsage, MemoryUsage) {
	newLeafNodes, newBranchNodes := atreeNodes(count, elementSize)
	if array {
		return MemoryUsage{
				Kind:   MemoryKindAtreeArrayDataSlab,
				Amount: newLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeArrayMetaDataSlab,
				Amount: newBranchNodes,
			}
	} else {
		return MemoryUsage{
				Kind:   MemoryKindAtreeMapDataSlab,
				Amount: newLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeMapMetaDataSlab,
				Amount: newBranchNodes,
			}
	}
}

func NewCadenceArrayMemoryUsages(length int) (MemoryUsage, MemoryUsage) {
	return CadenceArrayValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindCadenceArrayValueLength,
		Amount: uint64(length),
	}
}

func AdditionalAtreeMemoryUsage(originalCount uint64, elementSize uint, array bool) (MemoryUsage, MemoryUsage) {
	originalLeafNodes, originalBranchNodes := atreeNodes(originalCount, elementSize)
	newLeafNodes, newBranchNodes := atreeNodes(originalCount+1, elementSize)
	if array {
		return MemoryUsage{
				Kind:   MemoryKindAtreeArrayDataSlab,
				Amount: newLeafNodes - originalLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeArrayMetaDataSlab,
				Amount: newBranchNodes - originalBranchNodes,
			}
	} else {
		return MemoryUsage{
				Kind:   MemoryKindAtreeMapDataSlab,
				Amount: newLeafNodes - originalLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeMapMetaDataSlab,
				Amount: newBranchNodes - originalBranchNodes,
			}
	}
}

func NewArrayMemoryUsages(count uint64, elementSize uint) (MemoryUsage, MemoryUsage, MemoryUsage, MemoryUsage) {
	leaves, branches := newAtreeMemoryUsage(count, elementSize, true)
	return ArrayValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindAtreeArrayElementOverhead,
		Amount: count,
	}, leaves, branches
}

func NewDictionaryMemoryUsages(count uint64, elementSize uint) (MemoryUsage, MemoryUsage, MemoryUsage, MemoryUsage) {
	leaves, branches := newAtreeMemoryUsage(count, elementSize, false)
	return DictionaryValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindAtreeMapElementOverhead,
		Amount: count,
	}, leaves, branches
}

func NewCompositeMemoryUsages(count uint64, elementSize uint) (MemoryUsage, MemoryUsage, MemoryUsage, MemoryUsage) {
	leaves, branches := newAtreeMemoryUsage(count, elementSize, false)
	return CompositeValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindAtreeMapElementOverhead,
		Amount: count,
	}, leaves, branches
}

func NewAtreeMapPreAllocatedElementsMemoryUsage(count uint64, elementSize uint) MemoryUsage {
	leafNodesCount, _ := atreeNodes(count, elementSize)

	const preAllocatedElementCountPerLeafNode uint64 = 32

	var amount uint64 = 0
	if count < preAllocatedElementCountPerLeafNode*leafNodesCount {
		amount = preAllocatedElementCountPerLeafNode*leafNodesCount - count
	}

	return MemoryUsage{
		Kind:   MemoryKindAtreeMapPreAllocatedElement,
		Amount: amount,
	}
}

func NewSimpleCompositeMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindSimpleCompositeValue,
		Amount: uint64(length),
	}
}

func NewStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindStringValue,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewRawStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindRawString,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewCadenceStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceStringValue,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewCadenceCharacterMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceCharacterValue,
		Amount: uint64(length),
	}
}

func NewCadenceIntMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceIntValue,
		Amount: uint64(bytes),
	}
}

func NewBytesMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindBytes,
		Amount: uint64(length) + 1, // +1 to account for empty arrays
	}
}

func NewCharacterMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCharacterValue,
		Amount: uint64(length),
	}
}

func NewBigIntMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindBigInt,
		Amount: uint64(bytes),
	}
}

func NewCadenceStructMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return CadenceStructValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindCadenceStructValueSize,
		Amount: uint64(fields),
	}
}

func NewCadenceResourceMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return CadenceResourceValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindCadenceResourceValueSize,
		Amount: uint64(fields),
	}
}

func NewCadenceEventMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return CadenceEventValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindCadenceEventValueSize,
		Amount: uint64(fields),
	}
}

func NewCadenceContractMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return CadenceContractValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindCadenceContractValueSize,
		Amount: uint64(fields),
	}
}

func NewCadenceEnumMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return CadenceEnumValueBaseMemoryUsage, MemoryUsage{
		Kind:   MemoryKindCadenceEnumValueSize,
		Amount: uint64(fields),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const BigIntWordSize = int(unsafe.Sizeof(big.Word(0)))

var bigIntWordSizeAsBig = big.NewInt(int64(BigIntWordSize))

func BigIntByteLength(v *big.Int) int {
	// NOTE: big.Int.Bits() actually returns a slice of words,
	// []big.Word, where big.Word = uint,
	// NOT a slice of bytes!
	return len(v.Bits()) * BigIntWordSize
}

// big.Int memory metering:
// - |x| is len(x.Bits()), which is the length in words
//

func NewPlusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if |a|==|b|==0, then 0
	// else max(|a|, |b|) + 5
	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	if aWordLength == 0 && bWordLength == 0 {
		return NewBigIntMemoryUsage(0)
	}

	maxWordLength := max(
		aWordLength,
		bWordLength,
	)
	return NewBigIntMemoryUsage(
		(maxWordLength + 5) *
			BigIntWordSize,
	)
}

func NewMinusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// max(|a|, |b|) + 4

	maxWordLength := max(
		len(a.Bits()),
		len(b.Bits()),
	)
	return NewBigIntMemoryUsage(
		(maxWordLength + 4) *
			BigIntWordSize,
	)
}

func NewMulBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if min(|a|, |b|) <= 40:
	//     |a| + |b| + 4
	// else:
	//     n = min(|a|, |b|)
	//     3 * n + max(6 * n, |a| + |b|) + 8

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())
	minWordLength := min(
		aWordLength,
		bWordLength,
	)
	wordLengthSum := aWordLength + bWordLength

	var resultWordLength int
	if minWordLength <= 40 {
		resultWordLength = wordLengthSum + 4
	} else {
		resultWordLength = 3*minWordLength + max(6*minWordLength, wordLengthSum) + 8
	}
	return NewBigIntMemoryUsage(
		resultWordLength * BigIntWordSize,
	)
}

func NewModBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a < b or |b| == 1:
	//     |a| + 4
	// else if |b| < 100:
	//     |a| - |b| + 5
	// else:
	//     recursion_cost = pointer_size + 9 * |b| + floor(|a| / |b|) + 12
	//     recursion_depth = 2 * BitLen(b)
	//     3 * |b| + 4 + recursion_cost * recursion_depth

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Cmp(b) < 0 || bWordLength == 1 {
		resultWordLength = aWordLength + 4
	} else if bWordLength < 100 {
		resultWordLength = aWordLength - bWordLength + 5
	} else {
		recursionCost := int(unsafe.Sizeof(uintptr(0))) +
			9*bWordLength +
			aWordLength/bWordLength + 12
		recursionDepth := 2 * b.BitLen()
		resultWordLength = 3*bWordLength + 4 + recursionCost*recursionDepth
	}
	return NewBigIntMemoryUsage(
		resultWordLength * BigIntWordSize,
	)
}

func NewDivBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewModBigIntMemoryUsage(a, b)
}

func NewBitwiseOrBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0 and b >= 0:
	//     max(|a|, |b|) + 4
	// else if a <= 0 and b <= 0:
	//     |a| + |b| + min(|a|, |b|) + 13
	// else:
	//     2 * max(|a|, |b|) + 9

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Sign() >= 0 && b.Sign() >= 0 {
		resultWordLength = max(aWordLength, bWordLength) + 4
	} else if a.Sign() <= 0 && b.Sign() <= 0 {
		resultWordLength = aWordLength + bWordLength + min(aWordLength, bWordLength) + 13
	} else {
		resultWordLength = 2*max(aWordLength, bWordLength) + 9
	}
	return NewBigIntMemoryUsage(
		resultWordLength * BigIntWordSize,
	)
}

func NewBitwiseXorBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0 and b >= 0:
	//     max(|a|, |b|) + 4
	// else if a <= 0 and b <= 0:
	//     |a| + |b| + min(|a|, |b|) + 12
	// else:
	//     2 * max(|a|, |b|) + 9

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Sign() >= 0 && b.Sign() >= 0 {
		resultWordLength = max(aWordLength, bWordLength) + 4
	} else if a.Sign() <= 0 && b.Sign() <= 0 {
		resultWordLength = aWordLength + bWordLength + min(aWordLength, bWordLength) + 12
	} else {
		resultWordLength = 2*max(aWordLength, bWordLength) + 9
	}
	return NewBigIntMemoryUsage(
		resultWordLength * BigIntWordSize,
	)
}

func NewBitwiseAndBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0 and b >= 0:
	//     max(|a|, |b|) + 4
	// else if a <= 0 and b <= 0:
	//     |a| + |b| + max(|a|, |b|) + 13
	// else:
	//     2 * max(|a|, |b|) + 8

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Sign() >= 0 && b.Sign() >= 0 {
		resultWordLength = max(aWordLength, bWordLength) + 4
	} else if a.Sign() <= 0 && b.Sign() <= 0 {
		resultWordLength = aWordLength + bWordLength + max(aWordLength, bWordLength) + 13
	} else {
		resultWordLength = 2*max(aWordLength, bWordLength) + 8
	}
	return NewBigIntMemoryUsage(
		resultWordLength * BigIntWordSize,
	)
}

var invalidLeftShift = errors.NewDefaultUserError("invalid left shift of non-Int64")

func NewBitwiseLeftShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if b == 0:
	//     |a| + 4
	// else:
	//     |a| + b/word_size + 5

	aWordLength := len(a.Bits())

	var resultWordLength int
	if b.Sign() == 0 {
		resultWordLength = aWordLength + 4
	} else {
		// TODO: meter the allocation of the metering itself
		shiftByteLengthBig := new(big.Int).Div(b, bigIntWordSizeAsBig)
		// TODO: handle big int shifts
		if !shiftByteLengthBig.IsInt64() {
			panic(invalidLeftShift)
		}
		shiftByteLength := int(shiftByteLengthBig.Int64())
		resultWordLength = aWordLength + shiftByteLength + 5
	}
	return NewBigIntMemoryUsage(
		resultWordLength * BigIntWordSize,
	)
}

func NewBitwiseRightShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0:
	//     if b == 0:
	//         |a| + 4
	//     else:
	//         |a| - b/word_size + 4
	// else:
	//     |a| + 4

	aWordLength := len(a.Bits())

	var resultWordLength int
	if a.Sign() >= 0 {
		if b.Sign() == 0 {
			resultWordLength = aWordLength + 4
		} else {
			// TODO: meter the allocation of the metering itself
			shiftByteLengthBig := new(big.Int).Div(b, bigIntWordSizeAsBig)
			// TODO: handle big int shifts
			if !shiftByteLengthBig.IsInt64() {
				panic(invalidLeftShift)
			}
			shiftByteLength := int(shiftByteLengthBig.Int64())
			resultWordLength = aWordLength - shiftByteLength + 4
		}
	} else {
		resultWordLength = aWordLength + 4
	}
	return NewBigIntMemoryUsage(
		resultWordLength * BigIntWordSize,
	)
}

func NewNegateBigIntMemoryUsage(b *big.Int) MemoryUsage {
	// |a| + 4

	return NewBigIntMemoryUsage(
		(len(b.Bits()) + 4) * BigIntWordSize,
	)
}

func NewNumberMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindNumberValue,
		Amount: uint64(bytes),
	}
}

func NewArrayExpressionMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind: MemoryKindArrayExpression,
		// +1 to account for empty arrays
		Amount: uint64(length) + 1,
	}
}

func NewDictionaryExpressionMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind: MemoryKindDictionaryExpression,
		// +1 to account for empty dictionaries
		Amount: uint64(length) + 1,
	}
}

func NewMembersMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind: MemoryKindMembers,
		// +1 to account for empty members
		Amount: uint64(length) + 1,
	}
}

func NewCadenceNumberMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceNumberValue,
		Amount: uint64(bytes),
	}
}

func NewCadenceBigIntMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceNumberValue,
		Amount: uint64(bytes),
	}
}

func NewOrderedMapMemoryUsages(size uint64) (MemoryUsage, MemoryUsage, MemoryUsage) {
	return OrderedMapMemoryUsage,
		MemoryUsage{
			Kind:   MemoryKindOrderedMapEntryList,
			Amount: size,
		},
		MemoryUsage{
			Kind:   MemoryKindOrderedMapEntry,
			Amount: size,
		}
}

func NewAtreeEncodedSlabMemoryUsage(slabsCount uint) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindAtreeEncodedSlab,
		Amount: uint64(slabsCount),
	}
}
