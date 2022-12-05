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

type NumberConversionArgumentTypes struct {
	Type  Type
	Range ast.Range
}

type CastingExpressionTypes struct {
	StaticValueType Type
	TargetType      Type
}

type Elaboration struct {
	lock                             *sync.RWMutex
	functionDeclarationFunctionTypes map[*ast.FunctionDeclaration]*FunctionType
	variableDeclarationTypes         map[*ast.VariableDeclaration]VariableDeclarationTypes
	assignmentStatementTypes         map[*ast.AssignmentStatement]AssignmentStatementTypes
	compositeDeclarationTypes        map[*ast.CompositeDeclaration]*CompositeType
	compositeTypeDeclarations        map[*CompositeType]*ast.CompositeDeclaration
	interfaceDeclarationTypes        map[*ast.InterfaceDeclaration]*InterfaceType
	interfaceTypeDeclarations        map[*InterfaceType]*ast.InterfaceDeclaration
	constructorFunctionTypes         map[*ast.SpecialFunctionDeclaration]*FunctionType
	functionExpressionFunctionTypes  map[*ast.FunctionExpression]*FunctionType
	invocationExpressionTypes        map[*ast.InvocationExpression]InvocationExpressionTypes
	castingExpressionTypes           map[*ast.CastingExpression]CastingExpressionTypes
	returnStatementTypes             map[*ast.ReturnStatement]ReturnStatementTypes
	binaryExpressionTypes            map[*ast.BinaryExpression]BinaryExpressionTypes
	memberExpressionMemberInfos      map[*ast.MemberExpression]MemberInfo
	memberExpressionExpectedTypes    map[*ast.MemberExpression]Type
	arrayExpressionTypes             map[*ast.ArrayExpression]ArrayExpressionTypes
	dictionaryExpressionTypes        map[*ast.DictionaryExpression]DictionaryExpressionTypes
	integerExpressionTypes           map[*ast.IntegerExpression]Type
	stringExpressionTypes            map[*ast.StringExpression]Type
	fixedPointExpressionTypes        map[*ast.FixedPointExpression]Type
	transactionDeclarationTypes      map[*ast.TransactionDeclaration]*TransactionType
	swapStatementTypes               map[*ast.SwapStatement]SwapStatementTypes
	// nestedResourceMoveExpressions indicates the index or member expression
	// is implicitly moving a resource out of the container, e.g. in a shift or swap statement.
	nestedResourceMoveExpressions       map[ast.Expression]struct{}
	CompositeNestedDeclarations         map[*ast.CompositeDeclaration]map[string]ast.Declaration
	InterfaceNestedDeclarations         map[*ast.InterfaceDeclaration]map[string]ast.Declaration
	PostConditionsRewrite               map[*ast.Conditions]PostConditionsRewrite
	EmitStatementEventTypes             map[*ast.EmitStatement]*CompositeType
	CompositeTypes                      map[TypeID]*CompositeType
	InterfaceTypes                      map[TypeID]*InterfaceType
	IdentifierInInvocationTypes         map[*ast.IdentifierExpression]Type
	ImportDeclarationsResolvedLocations map[*ast.ImportDeclaration][]ResolvedLocation
	globalValues                        *StringVariableOrderedMap
	globalTypes                         *StringVariableOrderedMap
	TransactionTypes                    []*TransactionType
	isChecking                          bool
	ReferenceExpressionBorrowTypes      map[*ast.ReferenceExpression]Type
	IndexExpressionTypes                map[*ast.IndexExpression]IndexExpressionTypes
	ForceExpressionTypes                map[*ast.ForceExpression]Type
	StaticCastTypes                     map[*ast.CastingExpression]CastTypes
	NumberConversionArgumentTypes       map[ast.Expression]NumberConversionArgumentTypes
	RuntimeCastTypes                    map[*ast.CastingExpression]RuntimeCastTypes
}

func NewElaboration(gauge common.MemoryGauge, extendedElaboration bool) *Elaboration {
	common.UseMemory(gauge, common.ElaborationMemoryUsage)
	elaboration := &Elaboration{
		lock:                                new(sync.RWMutex),
		CompositeNestedDeclarations:         map[*ast.CompositeDeclaration]map[string]ast.Declaration{},
		InterfaceNestedDeclarations:         map[*ast.InterfaceDeclaration]map[string]ast.Declaration{},
		PostConditionsRewrite:               map[*ast.Conditions]PostConditionsRewrite{},
		EmitStatementEventTypes:             map[*ast.EmitStatement]*CompositeType{},
		CompositeTypes:                      map[TypeID]*CompositeType{},
		InterfaceTypes:                      map[TypeID]*InterfaceType{},
		IdentifierInInvocationTypes:         map[*ast.IdentifierExpression]Type{},
		ImportDeclarationsResolvedLocations: map[*ast.ImportDeclaration][]ResolvedLocation{},
		ReferenceExpressionBorrowTypes:      map[*ast.ReferenceExpression]Type{},
		IndexExpressionTypes:                map[*ast.IndexExpression]IndexExpressionTypes{},
	}
	if extendedElaboration {
		elaboration.ForceExpressionTypes = map[*ast.ForceExpression]Type{}
		elaboration.StaticCastTypes = map[*ast.CastingExpression]CastTypes{}
		elaboration.RuntimeCastTypes = map[*ast.CastingExpression]RuntimeCastTypes{}
		elaboration.NumberConversionArgumentTypes = map[ast.Expression]NumberConversionArgumentTypes{}
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
func (e *Elaboration) FunctionEntryPointType() (*FunctionType, error) {

	entryPointValue, ok := e.GetGlobalValue(FunctionEntryPointName)
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

func (e *Elaboration) FunctionDeclarationFunctionType(declaration *ast.FunctionDeclaration) *FunctionType {
	if e.functionDeclarationFunctionTypes == nil {
		return nil
	}
	return e.functionDeclarationFunctionTypes[declaration]
}

func (e *Elaboration) SetFunctionDeclarationFunctionType(
	declaration *ast.FunctionDeclaration,
	functionType *FunctionType,
) {
	if e.functionDeclarationFunctionTypes == nil {
		e.functionDeclarationFunctionTypes = map[*ast.FunctionDeclaration]*FunctionType{}
	}
	e.functionDeclarationFunctionTypes[declaration] = functionType
}

func (e *Elaboration) VariableDeclarationTypes(declaration *ast.VariableDeclaration) (types VariableDeclarationTypes) {
	if e.variableDeclarationTypes == nil {
		return
	}
	return e.variableDeclarationTypes[declaration]
}

func (e *Elaboration) SetVariableDeclarationTypes(
	declaration *ast.VariableDeclaration,
	types VariableDeclarationTypes,
) {
	if e.variableDeclarationTypes == nil {
		e.variableDeclarationTypes = map[*ast.VariableDeclaration]VariableDeclarationTypes{}
	}
	e.variableDeclarationTypes[declaration] = types
}

func (e *Elaboration) VariableDeclarationTypesCount() int {
	return len(e.variableDeclarationTypes)
}

func (e *Elaboration) AssignmentStatementTypes(assignment *ast.AssignmentStatement) (types AssignmentStatementTypes) {
	if e.assignmentStatementTypes == nil {
		return
	}
	return e.assignmentStatementTypes[assignment]
}

func (e *Elaboration) SetAssignmentStatementTypes(
	assignment *ast.AssignmentStatement,
	types AssignmentStatementTypes,
) {
	if e.assignmentStatementTypes == nil {
		e.assignmentStatementTypes = map[*ast.AssignmentStatement]AssignmentStatementTypes{}
	}
	e.assignmentStatementTypes[assignment] = types
}

func (e *Elaboration) CompositeDeclarationType(declaration *ast.CompositeDeclaration) *CompositeType {
	if e.compositeDeclarationTypes == nil {
		return nil
	}
	return e.compositeDeclarationTypes[declaration]
}

func (e *Elaboration) SetCompositeDeclarationType(
	declaration *ast.CompositeDeclaration,
	compositeType *CompositeType,
) {
	if e.compositeDeclarationTypes == nil {
		e.compositeDeclarationTypes = map[*ast.CompositeDeclaration]*CompositeType{}
	}
	e.compositeDeclarationTypes[declaration] = compositeType
}

func (e *Elaboration) CompositeTypeDeclaration(compositeType *CompositeType) *ast.CompositeDeclaration {
	if e.compositeTypeDeclarations == nil {
		return nil
	}
	return e.compositeTypeDeclarations[compositeType]
}

func (e *Elaboration) SetCompositeTypeDeclaration(
	compositeType *CompositeType,
	declaration *ast.CompositeDeclaration,
) {
	if e.compositeTypeDeclarations == nil {
		e.compositeTypeDeclarations = map[*CompositeType]*ast.CompositeDeclaration{}
	}
	e.compositeTypeDeclarations[compositeType] = declaration
}

func (e *Elaboration) InterfaceDeclarationType(declaration *ast.InterfaceDeclaration) *InterfaceType {
	if e.interfaceDeclarationTypes == nil {
		return nil
	}
	return e.interfaceDeclarationTypes[declaration]
}

func (e *Elaboration) SetInterfaceDeclarationType(
	declaration *ast.InterfaceDeclaration,
	interfaceType *InterfaceType,
) {
	if e.interfaceDeclarationTypes == nil {
		e.interfaceDeclarationTypes = map[*ast.InterfaceDeclaration]*InterfaceType{}
	}
	e.interfaceDeclarationTypes[declaration] = interfaceType
}

func (e *Elaboration) InterfaceTypeDeclaration(interfaceType *InterfaceType) *ast.InterfaceDeclaration {
	if e.interfaceTypeDeclarations == nil {
		return nil
	}
	return e.interfaceTypeDeclarations[interfaceType]
}

func (e *Elaboration) SetInterfaceTypeDeclaration(
	interfaceType *InterfaceType,
	declaration *ast.InterfaceDeclaration,
) {
	if e.interfaceTypeDeclarations == nil {
		e.interfaceTypeDeclarations = map[*InterfaceType]*ast.InterfaceDeclaration{}
	}
	e.interfaceTypeDeclarations[interfaceType] = declaration
}

func (e *Elaboration) ConstructorFunctionType(initializer *ast.SpecialFunctionDeclaration) *FunctionType {
	if e.constructorFunctionTypes == nil {
		return nil
	}
	return e.constructorFunctionTypes[initializer]
}

func (e *Elaboration) SetConstructorFunctionType(
	initializer *ast.SpecialFunctionDeclaration,
	functionType *FunctionType,
) {
	if e.constructorFunctionTypes == nil {
		e.constructorFunctionTypes = map[*ast.SpecialFunctionDeclaration]*FunctionType{}
	}
	e.constructorFunctionTypes[initializer] = functionType
}

func (e *Elaboration) FunctionExpressionFunctionType(expression *ast.FunctionExpression) *FunctionType {
	if e.functionExpressionFunctionTypes == nil {
		return nil
	}
	return e.functionExpressionFunctionTypes[expression]
}

func (e *Elaboration) SetFunctionExpressionFunctionType(
	expression *ast.FunctionExpression,
	functionType *FunctionType,
) {
	if e.functionExpressionFunctionTypes == nil {
		e.functionExpressionFunctionTypes = map[*ast.FunctionExpression]*FunctionType{}
	}
	e.functionExpressionFunctionTypes[expression] = functionType
}

func (e *Elaboration) InvocationExpressionTypes(
	expression *ast.InvocationExpression,
) (types InvocationExpressionTypes) {
	if e.invocationExpressionTypes == nil {
		return
	}
	return e.invocationExpressionTypes[expression]
}

func (e *Elaboration) SetInvocationExpressionTypes(
	expression *ast.InvocationExpression,
	types InvocationExpressionTypes,
) {
	if e.invocationExpressionTypes == nil {
		e.invocationExpressionTypes = map[*ast.InvocationExpression]InvocationExpressionTypes{}
	}
	e.invocationExpressionTypes[expression] = types
}

func (e *Elaboration) CastingExpressionTypes(expression *ast.CastingExpression) (types CastingExpressionTypes) {
	if e.castingExpressionTypes == nil {
		return
	}
	return e.castingExpressionTypes[expression]
}

func (e *Elaboration) SetCastingExpressionTypes(
	expression *ast.CastingExpression,
	types CastingExpressionTypes,
) {
	if e.castingExpressionTypes == nil {
		e.castingExpressionTypes = map[*ast.CastingExpression]CastingExpressionTypes{}
	}
	e.castingExpressionTypes[expression] = types
}

var defaultElaborationStringExpressionType = StringType

func (e *Elaboration) StringExpressionType(expression *ast.StringExpression) Type {
	if e.stringExpressionTypes != nil {
		result, ok := e.stringExpressionTypes[expression]
		if ok {
			return result
		}
	}
	// default, Elaboration.SetStringExpressionType
	return defaultElaborationStringExpressionType
}

func (e *Elaboration) SetStringExpressionType(expression *ast.StringExpression, ty Type) {
	if ty == defaultElaborationStringExpressionType {
		// default, see Elaboration.StringExpressionType
		return
	}
	if e.stringExpressionTypes == nil {
		e.stringExpressionTypes = map[*ast.StringExpression]Type{}
	}
	e.stringExpressionTypes[expression] = ty
}

func (e *Elaboration) ReturnStatementTypes(statement *ast.ReturnStatement) (types ReturnStatementTypes) {
	if e.returnStatementTypes == nil {
		return
	}
	return e.returnStatementTypes[statement]
}

func (e *Elaboration) SetReturnStatementTypes(statement *ast.ReturnStatement, types ReturnStatementTypes) {
	if e.returnStatementTypes == nil {
		e.returnStatementTypes = map[*ast.ReturnStatement]ReturnStatementTypes{}
	}
	e.returnStatementTypes[statement] = types
}

func (e *Elaboration) BinaryExpressionTypes(expression *ast.BinaryExpression) (types BinaryExpressionTypes) {
	if e.binaryExpressionTypes == nil {
		return
	}
	return e.binaryExpressionTypes[expression]
}

func (e *Elaboration) SetBinaryExpressionTypes(expression *ast.BinaryExpression, types BinaryExpressionTypes) {
	if e.binaryExpressionTypes == nil {
		e.binaryExpressionTypes = map[*ast.BinaryExpression]BinaryExpressionTypes{}
	}
	e.binaryExpressionTypes[expression] = types
}

func (e *Elaboration) IsNestedResourceMoveExpression(expression ast.Expression) bool {
	if e.nestedResourceMoveExpressions == nil {
		return false
	}
	_, ok := e.nestedResourceMoveExpressions[expression]
	return ok
}

func (e *Elaboration) SetIsNestedResourceMoveExpression(expression ast.Expression) {
	if e.nestedResourceMoveExpressions == nil {
		e.nestedResourceMoveExpressions = map[ast.Expression]struct{}{}
	}
	e.nestedResourceMoveExpressions[expression] = struct{}{}
}

func (e *Elaboration) GetGlobalType(name string) (*Variable, bool) {
	if e.globalTypes == nil {
		return nil, false
	}
	return e.globalTypes.Get(name)
}

func (e *Elaboration) GetGlobalValue(name string) (*Variable, bool) {
	if e.globalValues == nil {
		return nil, false
	}
	return e.globalValues.Get(name)
}

func (e *Elaboration) ForEachGlobalType(f func(name string, variable *Variable)) {
	if e.globalTypes == nil {
		return
	}
	e.globalTypes.Foreach(f)
}

func (e *Elaboration) ForEachGlobalValue(f func(name string, variable *Variable)) {
	if e.globalValues == nil {
		return
	}
	e.globalValues.Foreach(f)
}

func (e *Elaboration) SetGlobalValue(name string, variable *Variable) {
	if e.globalValues == nil {
		e.globalValues = &StringVariableOrderedMap{}
	}
	e.globalValues.Set(name, variable)
}

func (e *Elaboration) SetGlobalType(name string, variable *Variable) {
	if e.globalTypes == nil {
		e.globalTypes = &StringVariableOrderedMap{}
	}
	e.globalTypes.Set(name, variable)
}

func (e *Elaboration) ArrayExpressionTypes(expression *ast.ArrayExpression) (types ArrayExpressionTypes) {
	if e.arrayExpressionTypes == nil {
		return
	}
	return e.arrayExpressionTypes[expression]
}

func (e *Elaboration) SetArrayExpressionTypes(expression *ast.ArrayExpression, types ArrayExpressionTypes) {
	if e.arrayExpressionTypes == nil {
		e.arrayExpressionTypes = map[*ast.ArrayExpression]ArrayExpressionTypes{}
	}
	e.arrayExpressionTypes[expression] = types
}

func (e *Elaboration) DictionaryExpressionTypes(
	expression *ast.DictionaryExpression,
) (types DictionaryExpressionTypes) {
	if e.dictionaryExpressionTypes == nil {
		return
	}
	return e.dictionaryExpressionTypes[expression]
}

func (e *Elaboration) SetDictionaryExpressionTypes(
	expression *ast.DictionaryExpression,
	types DictionaryExpressionTypes,
) {
	if e.dictionaryExpressionTypes == nil {
		e.dictionaryExpressionTypes = map[*ast.DictionaryExpression]DictionaryExpressionTypes{}
	}
	e.dictionaryExpressionTypes[expression] = types
}

var defaultElaborationIntegerExpressionType = IntType

func (e *Elaboration) IntegerExpressionType(expression *ast.IntegerExpression) Type {
	if e.integerExpressionTypes != nil {
		result, ok := e.integerExpressionTypes[expression]
		if ok {
			return result
		}
	}
	// default, see Elaboration.SetIntegerExpressionType
	return defaultElaborationIntegerExpressionType
}

func (e *Elaboration) SetIntegerExpressionType(expression *ast.IntegerExpression, actualType Type) {
	if actualType == defaultElaborationIntegerExpressionType {
		// default, see Elaboration.IntegerExpressionType
		return
	}
	if e.integerExpressionTypes == nil {
		e.integerExpressionTypes = map[*ast.IntegerExpression]Type{}
	}
	e.integerExpressionTypes[expression] = actualType
}

func (e *Elaboration) MemberExpressionMemberInfo(expression *ast.MemberExpression) (memberInfo MemberInfo, ok bool) {
	if e.memberExpressionMemberInfos == nil {
		ok = false
		return
	}
	memberInfo, ok = e.memberExpressionMemberInfos[expression]
	return
}

func (e *Elaboration) SetMemberExpressionMemberInfo(expression *ast.MemberExpression, memberInfo MemberInfo) {
	if e.memberExpressionMemberInfos == nil {
		e.memberExpressionMemberInfos = map[*ast.MemberExpression]MemberInfo{}
	}
	e.memberExpressionMemberInfos[expression] = memberInfo
}

func (e *Elaboration) MemberExpressionExpectedType(expression *ast.MemberExpression) Type {
	if e.memberExpressionExpectedTypes == nil {
		return nil
	}
	return e.memberExpressionExpectedTypes[expression]
}

func (e *Elaboration) SetMemberExpressionExpectedType(expression *ast.MemberExpression, expectedType Type) {
	if e.memberExpressionExpectedTypes == nil {
		e.memberExpressionExpectedTypes = map[*ast.MemberExpression]Type{}
	}
	e.memberExpressionExpectedTypes[expression] = expectedType
}

var defaultElaborationFixedPointExpressionType = UFix64Type

func (e *Elaboration) FixedPointExpression(expression *ast.FixedPointExpression) Type {
	if e.fixedPointExpressionTypes != nil {
		result, ok := e.fixedPointExpressionTypes[expression]
		if ok {
			return result
		}
	}
	// default, Elaboration.SetFixedPointExpressionType
	return defaultElaborationFixedPointExpressionType
}

func (e *Elaboration) SetFixedPointExpression(expression *ast.FixedPointExpression, ty Type) {
	if ty == defaultElaborationFixedPointExpressionType {
		// default, see Elaboration.FixedPointExpressionType
		return
	}
	if e.fixedPointExpressionTypes == nil {
		e.fixedPointExpressionTypes = map[*ast.FixedPointExpression]Type{}
	}
	e.fixedPointExpressionTypes[expression] = ty
}

func (e *Elaboration) TransactionDeclarationType(declaration *ast.TransactionDeclaration) *TransactionType {
	if e.transactionDeclarationTypes == nil {
		return nil
	}
	return e.transactionDeclarationTypes[declaration]
}

func (e *Elaboration) SetTransactionDeclarationType(declaration *ast.TransactionDeclaration, ty *TransactionType) {
	if e.transactionDeclarationTypes == nil {
		e.transactionDeclarationTypes = map[*ast.TransactionDeclaration]*TransactionType{}
	}
	e.transactionDeclarationTypes[declaration] = ty
}

func (e *Elaboration) SetSwapStatementTypes(statement *ast.SwapStatement, types SwapStatementTypes) {
	if e.swapStatementTypes == nil {
		e.swapStatementTypes = map[*ast.SwapStatement]SwapStatementTypes{}
	}
	e.swapStatementTypes[statement] = types
}

func (e *Elaboration) SwapStatementTypes(statement *ast.SwapStatement) (types SwapStatementTypes) {
	if e.swapStatementTypes == nil {
		return
	}
	return e.swapStatementTypes[statement]
}
