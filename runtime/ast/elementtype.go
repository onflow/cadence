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

package ast

//go:generate go run golang.org/x/tools/cmd/stringer -type=ElementType

type ElementType uint64

const (
	ElementTypeUnknown ElementType = iota

	ElementTypeProgram
	ElementTypeBlock
	ElementTypeFunctionBlock

	// Declarations

	ElementTypeFunctionDeclaration
	ElementTypeSpecialFunctionDeclaration
	ElementTypeCompositeDeclaration
	ElementTypeInterfaceDeclaration
	ElementTypeAttachmentDeclaration
	ElementTypeFieldDeclaration
	ElementTypeEnumCaseDeclaration
	ElementTypePragmaDeclaration
	ElementTypeImportDeclaration
	ElementTypeTransactionDeclaration

	// Statements

	ElementTypeReturnStatement
	ElementTypeBreakStatement
	ElementTypeContinueStatement
	ElementTypeIfStatement
	ElementTypeSwitchStatement
	ElementTypeWhileStatement
	ElementTypeForStatement
	ElementTypeEmitStatement
	ElementTypeVariableDeclaration
	ElementTypeAssignmentStatement
	ElementTypeSwapStatement
	ElementTypeExpressionStatement
	ElementTypeRemoveStatement

	// Expressions

	ElementTypeVoidExpression
	ElementTypeBoolExpression
	ElementTypeNilExpression
	ElementTypeIntegerExpression
	ElementTypeFixedPointExpression
	ElementTypeArrayExpression
	ElementTypeDictionaryExpression
	ElementTypeIdentifierExpression
	ElementTypeInvocationExpression
	ElementTypeMemberExpression
	ElementTypeIndexExpression
	ElementTypeConditionalExpression
	ElementTypeUnaryExpression
	ElementTypeBinaryExpression
	ElementTypeFunctionExpression
	ElementTypeStringExpression
	ElementTypeCastingExpression
	ElementTypeCreateExpression
	ElementTypeDestroyExpression
	ElementTypeReferenceExpression
	ElementTypeForceExpression
	ElementTypePathExpression
	ElementTypeAttachExpression
)
