/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
)

type MemberInfo struct {
	Member       *Member
	IsOptional   bool
	AccessedType Type
}

type Elaboration struct {
	lock                                   *sync.RWMutex
	FunctionDeclarationFunctionTypes       map[*ast.FunctionDeclaration]*FunctionType
	VariableDeclarationValueTypes          map[*ast.VariableDeclaration]Type
	VariableDeclarationSecondValueTypes    map[*ast.VariableDeclaration]Type
	VariableDeclarationTargetTypes         map[*ast.VariableDeclaration]Type
	AssignmentStatementValueTypes          map[*ast.AssignmentStatement]Type
	AssignmentStatementTargetTypes         map[*ast.AssignmentStatement]Type
	CompositeDeclarationTypes              map[*ast.CompositeDeclaration]*CompositeType
	SpecialFunctionTypes                   map[*ast.SpecialFunctionDeclaration]*SpecialFunctionType
	FunctionExpressionFunctionType         map[*ast.FunctionExpression]*FunctionType
	InvocationExpressionArgumentTypes      map[*ast.InvocationExpression][]Type
	InvocationExpressionParameterTypes     map[*ast.InvocationExpression][]Type
	InvocationExpressionReturnTypes        map[*ast.InvocationExpression]Type
	InterfaceDeclarationTypes              map[*ast.InterfaceDeclaration]*InterfaceType
	CastingStaticValueTypes                map[*ast.CastingExpression]Type
	CastingTargetTypes                     map[*ast.CastingExpression]Type
	ReturnStatementValueTypes              map[*ast.ReturnStatement]Type
	ReturnStatementReturnTypes             map[*ast.ReturnStatement]Type
	BinaryExpressionResultTypes            map[*ast.BinaryExpression]Type
	BinaryExpressionRightTypes             map[*ast.BinaryExpression]Type
	MemberExpressionMemberInfos            map[*ast.MemberExpression]MemberInfo
	ArrayExpressionArgumentTypes           map[*ast.ArrayExpression][]Type
	ArrayExpressionElementType             map[*ast.ArrayExpression]Type
	DictionaryExpressionType               map[*ast.DictionaryExpression]*DictionaryType
	DictionaryExpressionEntryTypes         map[*ast.DictionaryExpression][]DictionaryEntryType
	TransactionDeclarationTypes            map[*ast.TransactionDeclaration]*TransactionType
	SwapStatementLeftTypes                 map[*ast.SwapStatement]Type
	SwapStatementRightTypes                map[*ast.SwapStatement]Type
	IsResourceMovingStorageIndexExpression map[*ast.IndexExpression]bool
	CompositeNestedDeclarations            map[*ast.CompositeDeclaration]map[string]ast.Declaration
	InterfaceNestedDeclarations            map[*ast.InterfaceDeclaration]map[string]ast.Declaration
	PostConditionsRewrite                  map[*ast.Conditions]PostConditionsRewrite
	EmitStatementEventTypes                map[*ast.EmitStatement]*CompositeType
	// Keyed by qualified identifier
	CompositeTypes                      map[TypeID]*CompositeType
	InterfaceTypes                      map[TypeID]*InterfaceType
	InvocationExpressionTypeArguments   map[*ast.InvocationExpression]*TypeParameterTypeOrderedMap
	IdentifierInInvocationTypes         map[*ast.IdentifierExpression]Type
	ImportDeclarationsResolvedLocations map[*ast.ImportDeclaration][]ResolvedLocation
	GlobalValues                        *StringVariableOrderedMap
	GlobalTypes                         *StringVariableOrderedMap
	TransactionTypes                    []*TransactionType
	EffectivePredeclaredValues          map[string]ValueDeclaration
	EffectivePredeclaredTypes           map[string]TypeDeclaration
	isChecking                          bool
}

func NewElaboration() *Elaboration {
	return &Elaboration{
		lock:                                   new(sync.RWMutex),
		FunctionDeclarationFunctionTypes:       map[*ast.FunctionDeclaration]*FunctionType{},
		VariableDeclarationValueTypes:          map[*ast.VariableDeclaration]Type{},
		VariableDeclarationSecondValueTypes:    map[*ast.VariableDeclaration]Type{},
		VariableDeclarationTargetTypes:         map[*ast.VariableDeclaration]Type{},
		AssignmentStatementValueTypes:          map[*ast.AssignmentStatement]Type{},
		AssignmentStatementTargetTypes:         map[*ast.AssignmentStatement]Type{},
		CompositeDeclarationTypes:              map[*ast.CompositeDeclaration]*CompositeType{},
		SpecialFunctionTypes:                   map[*ast.SpecialFunctionDeclaration]*SpecialFunctionType{},
		FunctionExpressionFunctionType:         map[*ast.FunctionExpression]*FunctionType{},
		InvocationExpressionArgumentTypes:      map[*ast.InvocationExpression][]Type{},
		InvocationExpressionParameterTypes:     map[*ast.InvocationExpression][]Type{},
		InvocationExpressionReturnTypes:        map[*ast.InvocationExpression]Type{},
		InterfaceDeclarationTypes:              map[*ast.InterfaceDeclaration]*InterfaceType{},
		CastingStaticValueTypes:                map[*ast.CastingExpression]Type{},
		CastingTargetTypes:                     map[*ast.CastingExpression]Type{},
		ReturnStatementValueTypes:              map[*ast.ReturnStatement]Type{},
		ReturnStatementReturnTypes:             map[*ast.ReturnStatement]Type{},
		BinaryExpressionResultTypes:            map[*ast.BinaryExpression]Type{},
		BinaryExpressionRightTypes:             map[*ast.BinaryExpression]Type{},
		MemberExpressionMemberInfos:            map[*ast.MemberExpression]MemberInfo{},
		ArrayExpressionArgumentTypes:           map[*ast.ArrayExpression][]Type{},
		ArrayExpressionElementType:             map[*ast.ArrayExpression]Type{},
		DictionaryExpressionType:               map[*ast.DictionaryExpression]*DictionaryType{},
		DictionaryExpressionEntryTypes:         map[*ast.DictionaryExpression][]DictionaryEntryType{},
		TransactionDeclarationTypes:            map[*ast.TransactionDeclaration]*TransactionType{},
		SwapStatementLeftTypes:                 map[*ast.SwapStatement]Type{},
		SwapStatementRightTypes:                map[*ast.SwapStatement]Type{},
		IsResourceMovingStorageIndexExpression: map[*ast.IndexExpression]bool{},
		CompositeNestedDeclarations:            map[*ast.CompositeDeclaration]map[string]ast.Declaration{},
		InterfaceNestedDeclarations:            map[*ast.InterfaceDeclaration]map[string]ast.Declaration{},
		PostConditionsRewrite:                  map[*ast.Conditions]PostConditionsRewrite{},
		EmitStatementEventTypes:                map[*ast.EmitStatement]*CompositeType{},
		CompositeTypes:                         map[TypeID]*CompositeType{},
		InterfaceTypes:                         map[TypeID]*InterfaceType{},
		InvocationExpressionTypeArguments:      map[*ast.InvocationExpression]*TypeParameterTypeOrderedMap{},
		IdentifierInInvocationTypes:            map[*ast.IdentifierExpression]Type{},
		ImportDeclarationsResolvedLocations:    map[*ast.ImportDeclaration][]ResolvedLocation{},
		GlobalValues:                           NewStringVariableOrderedMap(),
		GlobalTypes:                            NewStringVariableOrderedMap(),
		EffectivePredeclaredValues:             map[string]ValueDeclaration{},
		EffectivePredeclaredTypes:              map[string]TypeDeclaration{},
	}
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

	invokableType, ok := entryPointValue.Type.(InvokableType)
	if !ok {
		return nil, &InvalidEntryPointTypeError{
			Type: entryPointValue.Type,
		}
	}

	functionType := invokableType.InvocationFunctionType()
	return functionType, nil
}
