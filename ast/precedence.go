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

//go:generate stringer -type=precedence

// precedence is the order of importance of expressions / operators.
// NOTE: this enumeration does *NOT* influence parsing,
// and should be kept in sync with the binding powers in the parser
type precedence uint

const (
	precedenceUnknown precedence = iota
	// precedenceTernary is the precedence of
	// - ConditionalExpression. right associative!
	precedenceTernary
	// precedenceLogicalOr is the precedence of
	// - BinaryExpression, with OperationOr
	precedenceLogicalOr
	// precedenceLogicalAnd is the precedence of
	// - BinaryExpression, with OperationAnd
	precedenceLogicalAnd
	// precedenceComparison is the precedence of
	// - BinaryExpression, with OperationEqual, OperationNotEqual,
	//   OperationLessEqual, OperationLess,
	//   OperationGreater, or OperationGreaterEqual
	precedenceComparison
	// precedenceNilCoalescing is the precedence of
	// - BinaryExpression, with OperationNilCoalesce. right associative!
	precedenceNilCoalescing
	// precedenceBitwiseOr is the precedence of
	// - BinaryExpression, with OperationBitwiseOr
	precedenceBitwiseOr
	// precedenceBitwiseXor is the precedence of
	// - BinaryExpression, with OperationBitwiseXor
	precedenceBitwiseXor
	// precedenceBitwiseAnd is the precedence of
	// - BinaryExpression, with OperationBitwiseAnd
	precedenceBitwiseAnd
	// precedenceBitwiseShift is the precedence of
	// - BinaryExpression, with OperationBitwiseLeftShift or OperationBitwiseRightShift
	precedenceBitwiseShift
	// precedenceAddition is the precedence of
	// - BinaryExpression, with OperationPlus or OperationMinus
	precedenceAddition
	// precedenceMultiplication is the precedence of
	// - BinaryExpression, with OperationMul, OperationMod, or OperationDiv
	precedenceMultiplication
	// precedenceCasting is the precedence of
	// - CastingExpression
	precedenceCasting
	// precedenceUnaryPrefix is the precedence of
	// - UnaryExpression
	// - CreateExpression
	// - DestroyExpression
	// - ReferenceExpression
	precedenceUnaryPrefix
	// precedenceUnaryPostfix is the precedence of
	// - ForceExpression
	precedenceUnaryPostfix
	// precedenceAccess is the precedence of
	// - InvocationExpression
	// - IndexExpression
	// - MemberExpression
	precedenceAccess
	// precedenceLiteral is the precedence of
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
	precedenceLiteral
)
