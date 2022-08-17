/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"sync"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type MemberInfo struct {
	Member       *Member
	IsOptional   bool
	AccessedType Type
}

type CastTypes struct {
	ExprActualType Type
	TargetType     Type
	ExpectedType   Type
}

type RuntimeCastTypes struct {
	Left  Type
	Right Type
}

type ReturnStatementTypes struct {
	ValueType  Type
	ReturnType Type
}

type BinaryExpressionTypes struct {
	ResultType Type
	LeftType   Type
	RightType  Type
}

type VariableDeclarationTypes struct {
	ValueType       Type
	SecondValueType Type
	TargetType      Type
}

type AssignmentStatementTypes struct {
	ValueType  Type
	TargetType Type
}

type InvocationExpressionTypes struct {
	ArgumentTypes      []Type
	TypeParameterTypes []Type
	ReturnType         Type
	TypeArguments      *TypeParameterTypeOrderedMap
}

type ArrayExpressionTypes struct {
	ArgumentTypes []Type
	ArrayType     ArrayType
}

type DictionaryExpressionTypes struct {
	DictionaryType *DictionaryType
	EntryTypes     []DictionaryEntryType
}

type SwapStatementTypes struct {
	LeftType  Type
	RightType Type
}

type IndexExpressionTypes struct {
	IndexedType  ValueIndexableType
	IndexingType Type
}

type Elaboration struct {
	lock                             *sync.RWMutex
	FunctionDeclarationFunctionTypes map[*ast.FunctionDeclaration]*FunctionType
	VariableDeclarationTypes         map[*ast.VariableDeclaration]VariableDeclarationTypes
	AssignmentStatementTypes         map[*ast.AssignmentStatement]AssignmentStatementTypes
	CompositeDeclarationTypes        map[*ast.CompositeDeclaration]*CompositeType
	CompositeTypeDeclarations        map[*CompositeType]*ast.CompositeDeclaration
	InterfaceDeclarationTypes        map[*ast.InterfaceDeclaration]*InterfaceType
	InterfaceTypeDeclarations        map[*InterfaceType]*ast.InterfaceDeclaration
	ConstructorFunctionTypes         map[*ast.SpecialFunctionDeclaration]*FunctionType
	FunctionExpressionFunctionType   map[*ast.FunctionExpression]*FunctionType
	InvocationExpressionTypes        map[*ast.InvocationExpression]InvocationExpressionTypes
	CastingStaticValueTypes          map[*ast.CastingExpression]Type
	CastingTargetTypes               map[*ast.CastingExpression]Type
	ReturnStatementTypes             map[*ast.ReturnStatement]ReturnStatementTypes
	BinaryExpressionTypes            map[*ast.BinaryExpression]BinaryExpressionTypes
	MemberExpressionMemberInfos      map[*ast.MemberExpression]MemberInfo
	MemberExpressionExpectedTypes    map[*ast.MemberExpression]Type
	ArrayExpressionTypes             map[*ast.ArrayExpression]ArrayExpressionTypes
	DictionaryExpressionTypes        map[*ast.DictionaryExpression]DictionaryExpressionTypes
	IntegerExpressionType            map[*ast.IntegerExpression]Type
	StringExpressionType             map[*ast.StringExpression]Type
	FixedPointExpression             map[*ast.FixedPointExpression]Type
	TransactionDeclarationTypes      map[*ast.TransactionDeclaration]*TransactionType
	SwapStatementTypes               map[*ast.SwapStatement]SwapStatementTypes
	// IsNestedResourceMoveExpression indicates if the access the index or member expression
	// is implicitly moving a resource out of the container, e.g. in a shift or swap statement.
	IsNestedResourceMoveExpression      map[ast.Expression]struct{}
	CompositeNestedDeclarations         map[*ast.CompositeDeclaration]map[string]ast.Declaration
	InterfaceNestedDeclarations         map[*ast.InterfaceDeclaration]map[string]ast.Declaration
	PostConditionsRewrite               map[*ast.Conditions]PostConditionsRewrite
	EmitStatementEventTypes             map[*ast.EmitStatement]*CompositeType
	CompositeTypes                      map[TypeID]*CompositeType
	InterfaceTypes                      map[TypeID]*InterfaceType
	IdentifierInInvocationTypes         map[*ast.IdentifierExpression]Type
	ImportDeclarationsResolvedLocations map[*ast.ImportDeclaration][]ResolvedLocation
	GlobalValues                        *StringVariableOrderedMap
	GlobalTypes                         *StringVariableOrderedMap
	TransactionTypes                    []*TransactionType
	EffectivePredeclaredValues          map[string]ValueDeclaration
	EffectivePredeclaredTypes           map[string]TypeDeclaration
	isChecking                          bool
	ReferenceExpressionBorrowTypes      map[*ast.ReferenceExpression]Type
	IndexExpressionTypes                map[*ast.IndexExpression]IndexExpressionTypes
	ForceExpressionTypes                map[*ast.ForceExpression]Type
	StaticCastTypes                     map[*ast.CastingExpression]CastTypes
	NumberConversionArgumentTypes       map[ast.Expression]struct {
		Type  Type
		Range ast.Range
	}
	RuntimeCastTypes map[*ast.CastingExpression]RuntimeCastTypes
}

func NewElaboration(gauge common.MemoryGauge, extendedElaboration bool) *Elaboration {
	common.UseMemory(gauge, common.ElaborationMemoryUsage)
	elaboration := &Elaboration{
		lock:                                new(sync.RWMutex),
		FunctionDeclarationFunctionTypes:    map[*ast.FunctionDeclaration]*FunctionType{},
		VariableDeclarationTypes:            map[*ast.VariableDeclaration]VariableDeclarationTypes{},
		AssignmentStatementTypes:            map[*ast.AssignmentStatement]AssignmentStatementTypes{},
		CompositeDeclarationTypes:           map[*ast.CompositeDeclaration]*CompositeType{},
		CompositeTypeDeclarations:           map[*CompositeType]*ast.CompositeDeclaration{},
		InterfaceDeclarationTypes:           map[*ast.InterfaceDeclaration]*InterfaceType{},
		InterfaceTypeDeclarations:           map[*InterfaceType]*ast.InterfaceDeclaration{},
		ConstructorFunctionTypes:            map[*ast.SpecialFunctionDeclaration]*FunctionType{},
		FunctionExpressionFunctionType:      map[*ast.FunctionExpression]*FunctionType{},
		InvocationExpressionTypes:           map[*ast.InvocationExpression]InvocationExpressionTypes{},
		CastingStaticValueTypes:             map[*ast.CastingExpression]Type{},
		CastingTargetTypes:                  map[*ast.CastingExpression]Type{},
		ReturnStatementTypes:                map[*ast.ReturnStatement]ReturnStatementTypes{},
		BinaryExpressionTypes:               map[*ast.BinaryExpression]BinaryExpressionTypes{},
		MemberExpressionMemberInfos:         map[*ast.MemberExpression]MemberInfo{},
		MemberExpressionExpectedTypes:       map[*ast.MemberExpression]Type{},
		ArrayExpressionTypes:                map[*ast.ArrayExpression]ArrayExpressionTypes{},
		DictionaryExpressionTypes:           map[*ast.DictionaryExpression]DictionaryExpressionTypes{},
		IntegerExpressionType:               map[*ast.IntegerExpression]Type{},
		StringExpressionType:                map[*ast.StringExpression]Type{},
		FixedPointExpression:                map[*ast.FixedPointExpression]Type{},
		TransactionDeclarationTypes:         map[*ast.TransactionDeclaration]*TransactionType{},
		SwapStatementTypes:                  map[*ast.SwapStatement]SwapStatementTypes{},
		IsNestedResourceMoveExpression:      map[ast.Expression]struct{}{},
		CompositeNestedDeclarations:         map[*ast.CompositeDeclaration]map[string]ast.Declaration{},
		InterfaceNestedDeclarations:         map[*ast.InterfaceDeclaration]map[string]ast.Declaration{},
		PostConditionsRewrite:               map[*ast.Conditions]PostConditionsRewrite{},
		EmitStatementEventTypes:             map[*ast.EmitStatement]*CompositeType{},
		CompositeTypes:                      map[TypeID]*CompositeType{},
		InterfaceTypes:                      map[TypeID]*InterfaceType{},
		IdentifierInInvocationTypes:         map[*ast.IdentifierExpression]Type{},
		ImportDeclarationsResolvedLocations: map[*ast.ImportDeclaration][]ResolvedLocation{},
		GlobalValues:                        &StringVariableOrderedMap{},
		GlobalTypes:                         &StringVariableOrderedMap{},
		EffectivePredeclaredValues:          map[string]ValueDeclaration{},
		EffectivePredeclaredTypes:           map[string]TypeDeclaration{},
		ReferenceExpressionBorrowTypes:      map[*ast.ReferenceExpression]Type{},
		IndexExpressionTypes:                map[*ast.IndexExpression]IndexExpressionTypes{},
	}
	if extendedElaboration {
		elaboration.ForceExpressionTypes = map[*ast.ForceExpression]Type{}
		elaboration.StaticCastTypes = map[*ast.CastingExpression]CastTypes{}
		elaboration.RuntimeCastTypes = map[*ast.CastingExpression]RuntimeCastTypes{}
		elaboration.NumberConversionArgumentTypes = map[ast.Expression]struct {
			Type  Type
			Range ast.Range
		}{}
	}
	return elaboration

}

func (e *Elaboration) IsChecking() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.isChecking
}

func (e *Elaboration) setIsChecking(isChecking bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.isChecking = isChecking
}

// FunctionEntryPointType returns the type of the entry point function declaration, if any.
//
// Returns an error if no valid entry point function declaration exists.
//
func (e *Elaboration) FunctionEntryPointType() (*FunctionType, error) {

	entryPointValue, ok := e.GlobalValues.Get(FunctionEntryPointName)
	if !ok {
		return nil, &MissingEntryPointError{
			Expected: FunctionEntryPointName,
		}
	}

	functionType, ok := entryPointValue.Type.(*FunctionType)
	if !ok {
		return nil, &InvalidEntryPointTypeError{
			Type: entryPointValue.Type,
		}
	}

	return functionType, nil
}
