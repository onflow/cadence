/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

//go:generate stringer -type=expressionPrecedence
//go:generate stringer -type=TypePrecedence

// expressionPrecedence is the order of importance of expressions / operators.
// NOTE: this enumeration does *NOT* influence parsing,
// and should be kept in sync with the binding powers in the parser
type expressionPrecedence uint

const (
	expressionPrecedenceUnknown expressionPrecedence = iota
	// expressionPrecedenceTernary is the expressionPrecedence of
	// - ConditionalExpression. right associative!
	expressionPrecedenceTernary
	// expressionPrecedenceLogicalOr is the expressionPrecedence of
	// - BinaryExpression, with OperationOr
	expressionPrecedenceLogicalOr
	// expressionPrecedenceLogicalAnd is the expressionPrecedence of
	// - BinaryExpression, with OperationAnd
	expressionPrecedenceLogicalAnd
	// expressionPrecedenceComparison is the expressionPrecedence of
	// - BinaryExpression, with OperationEqual, OperationNotEqual,
	//   OperationLessEqual, OperationLess,
	//   OperationGreater, or OperationGreaterEqual
	expressionPrecedenceComparison
	// expressionPrecedenceNilCoalescing is the expressionPrecedence of
	// - BinaryExpression, with OperationNilCoalesce. right associative!
	expressionPrecedenceNilCoalescing
	// expressionPrecedenceBitwiseOr is the expressionPrecedence of
	// - BinaryExpression, with OperationBitwiseOr
	expressionPrecedenceBitwiseOr
	// expressionPrecedenceBitwiseXor is the expressionPrecedence of
	// - BinaryExpression, with OperationBitwiseXor
	expressionPrecedenceBitwiseXor
	// expressionPrecedenceBitwiseAnd is the expressionPrecedence of
	// - BinaryExpression, with OperationBitwiseAnd
	expressionPrecedenceBitwiseAnd
	// expressionPrecedenceBitwiseShift is the expressionPrecedence of
	// - BinaryExpression, with OperationBitwiseLeftShift or OperationBitwiseRightShift
	expressionPrecedenceBitwiseShift
	// expressionPrecedenceAddition is the expressionPrecedence of
	// - BinaryExpression, with OperationPlus or OperationMinus
	expressionPrecedenceAddition
	// expressionPrecedenceMultiplication is the expressionPrecedence of
	// - BinaryExpression, with OperationMul, OperationMod, or OperationDiv
	expressionPrecedenceMultiplication
	// expressionPrecedenceCasting is the expressionPrecedence of
	// - CastingExpression
	expressionPrecedenceCasting
	// expressionPrecedenceUnaryPrefix is the expressionPrecedence of
	// - UnaryExpression
	// - CreateExpression
	// - DestroyExpression
	// - ReferenceExpression
	expressionPrecedenceUnaryPrefix
	// expressionPrecedenceUnaryPostfix is the expressionPrecedence of
	// - ForceExpression
	expressionPrecedenceUnaryPostfix
	// expressionPrecedenceAccess is the expressionPrecedence of
	// - InvocationExpression
	// - IndexExpression
	// - MemberExpression
	expressionPrecedenceAccess
	// expressionPrecedenceLiteral is the expressionPrecedence of
	// - BoolExpression
	// - NilExpression
	// - StringExpression
	// - StringTemplateExpression
	// - IntegerExpression
	// - FixedPointExpression
	// - ArrayExpression
	// - DictionaryExpression
	// - IdentifierExpression
	// - FunctionExpression
	// - PathExpression
	// - AttachExpression
	expressionPrecedenceLiteral
)

// TypePrecedence is the order of importance of types / operators.
// NOTE: this enumeration does *NOT* influence parsing,
// and should be kept in sync with the binding powers in the parser
type TypePrecedence uint

const (
	TypePrecedenceUnknown TypePrecedence = iota
	// TypePrecedenceOptional is the TypePrecedence of OptionalType
	TypePrecedenceOptional
	// TypePrecedenceReference is the TypePrecedence of ReferenceType
	TypePrecedenceReference
	// TypePrecedenceInstantiation is the TypePrecedence of InstantiationType
	TypePrecedenceInstantiation
	// TypePrecedencePrimary is the TypePrecedence of all other types
	TypePrecedencePrimary
)
