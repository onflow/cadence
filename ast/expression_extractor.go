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

import (
	"fmt"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

type VoidExtractor interface {
	ExtractVoid(extractor *ExpressionExtractor, expression *VoidExpression) ExpressionExtraction
}
type BoolExtractor interface {
	ExtractBool(extractor *ExpressionExtractor, expression *BoolExpression) ExpressionExtraction
}

type NilExtractor interface {
	ExtractNil(extractor *ExpressionExtractor, expression *NilExpression) ExpressionExtraction
}

type IntExtractor interface {
	ExtractInteger(extractor *ExpressionExtractor, expression *IntegerExpression) ExpressionExtraction
}

type FixedPointExtractor interface {
	ExtractFixedPoint(extractor *ExpressionExtractor, expression *FixedPointExpression) ExpressionExtraction
}

type StringExtractor interface {
	ExtractString(extractor *ExpressionExtractor, expression *StringExpression) ExpressionExtraction
}

type StringTemplateExtractor interface {
	ExtractStringTemplate(extractor *ExpressionExtractor, expression *StringTemplateExpression) ExpressionExtraction
}

type ArrayExtractor interface {
	ExtractArray(extractor *ExpressionExtractor, expression *ArrayExpression) ExpressionExtraction
}

type DictionaryExtractor interface {
	ExtractDictionary(extractor *ExpressionExtractor, expression *DictionaryExpression) ExpressionExtraction
}

type IdentifierExtractor interface {
	ExtractIdentifier(extractor *ExpressionExtractor, expression *IdentifierExpression) ExpressionExtraction
}

type InvocationExtractor interface {
	ExtractInvocation(extractor *ExpressionExtractor, expression *InvocationExpression) ExpressionExtraction
}

type MemberExtractor interface {
	ExtractMember(extractor *ExpressionExtractor, expression *MemberExpression) ExpressionExtraction
}

type IndexExtractor interface {
	ExtractIndex(extractor *ExpressionExtractor, expression *IndexExpression) ExpressionExtraction
}

type ConditionalExtractor interface {
	ExtractConditional(extractor *ExpressionExtractor, expression *ConditionalExpression) ExpressionExtraction
}

type UnaryExtractor interface {
	ExtractUnary(extractor *ExpressionExtractor, expression *UnaryExpression) ExpressionExtraction
}

type BinaryExtractor interface {
	ExtractBinary(extractor *ExpressionExtractor, expression *BinaryExpression) ExpressionExtraction
}

type FunctionExtractor interface {
	ExtractFunction(extractor *ExpressionExtractor, expression *FunctionExpression) ExpressionExtraction
}

type CastingExtractor interface {
	ExtractCast(extractor *ExpressionExtractor, expression *CastingExpression) ExpressionExtraction
}

type CreateExtractor interface {
	ExtractCreate(extractor *ExpressionExtractor, expression *CreateExpression) ExpressionExtraction
}

type DestroyExtractor interface {
	ExtractDestroy(extractor *ExpressionExtractor, expression *DestroyExpression) ExpressionExtraction
}

type ReferenceExtractor interface {
	ExtractReference(extractor *ExpressionExtractor, expression *ReferenceExpression) ExpressionExtraction
}

type ForceExtractor interface {
	ExtractForce(extractor *ExpressionExtractor, expression *ForceExpression) ExpressionExtraction
}

type PathExtractor interface {
	ExtractPath(extractor *ExpressionExtractor, expression *PathExpression) ExpressionExtraction
}

type AttachExtractor interface {
	ExtractAttach(extractor *ExpressionExtractor, expression *AttachExpression) ExpressionExtraction
}

type ExpressionExtractor struct {
	IndexExtractor          IndexExtractor
	ForceExtractor          ForceExtractor
	BoolExtractor           BoolExtractor
	NilExtractor            NilExtractor
	IntExtractor            IntExtractor
	FixedPointExtractor     FixedPointExtractor
	StringExtractor         StringExtractor
	StringTemplateExtractor StringTemplateExtractor
	ArrayExtractor          ArrayExtractor
	DictionaryExtractor     DictionaryExtractor
	IdentifierExtractor     IdentifierExtractor
	AttachExtractor         AttachExtractor
	MemoryGauge             common.MemoryGauge
	VoidExtractor           VoidExtractor
	UnaryExtractor          UnaryExtractor
	ConditionalExtractor    ConditionalExtractor
	InvocationExtractor     InvocationExtractor
	BinaryExtractor         BinaryExtractor
	FunctionExtractor       FunctionExtractor
	CastingExtractor        CastingExtractor
	CreateExtractor         CreateExtractor
	DestroyExtractor        DestroyExtractor
	ReferenceExtractor      ReferenceExtractor
	MemberExtractor         MemberExtractor
	PathExtractor           PathExtractor
	nextIdentifier          int
}

var _ ExpressionVisitor[ExpressionExtraction] = &ExpressionExtractor{}

func (extractor *ExpressionExtractor) Extract(expression Expression) ExpressionExtraction {
	return AcceptExpression[ExpressionExtraction](expression, extractor)
}

func (extractor *ExpressionExtractor) FreshIdentifier() string {
	defer func() {
		extractor.nextIdentifier++
	}()
	// TODO: improve
	// NOTE: to avoid naming clashes with identifiers in the program,
	// include characters that can't be represented in source:
	//   - \x00 = Null character
	//   - \x1F = Information Separator One
	return extractor.FormatIdentifier(extractor.nextIdentifier)
}

func (extractor *ExpressionExtractor) FormatIdentifier(identifier int) string {
	return fmt.Sprintf("\x00exp\x1F%d", identifier)
}

type ExtractedExpression struct {
	Expression Expression
	Identifier Identifier
}

type ExpressionExtraction struct {
	RewrittenExpression  Expression
	ExtractedExpressions []ExtractedExpression
}

// utility for expressions whose rewritten form is identical, i.e. nothing to rewrite
func rewriteExpressionAsIs(expression Expression) ExpressionExtraction {
	return ExpressionExtraction{
		RewrittenExpression:  expression,
		ExtractedExpressions: nil,
	}
}

func (extractor *ExpressionExtractor) VisitVoidExpression(expression *VoidExpression) ExpressionExtraction {
	if extractor.VoidExtractor != nil {
		return extractor.VoidExtractor.ExtractVoid(extractor, expression)
	}

	return extractor.ExtractVoid(expression)
}

func (extractor *ExpressionExtractor) ExtractVoid(expression *VoidExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitBoolExpression(expression *BoolExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.BoolExtractor != nil {
		return extractor.BoolExtractor.ExtractBool(extractor, expression)
	}
	return extractor.ExtractBool(expression)
}

func (extractor *ExpressionExtractor) ExtractBool(expression *BoolExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitNilExpression(expression *NilExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.NilExtractor != nil {
		return extractor.NilExtractor.ExtractNil(extractor, expression)
	}
	return extractor.ExtractNil(expression)
}

func (extractor *ExpressionExtractor) ExtractNil(expression *NilExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitIntegerExpression(expression *IntegerExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.IntExtractor != nil {
		return extractor.IntExtractor.ExtractInteger(extractor, expression)
	}
	return extractor.ExtractInteger(expression)
}

func (extractor *ExpressionExtractor) ExtractInteger(expression *IntegerExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitFixedPointExpression(expression *FixedPointExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.FixedPointExtractor != nil {
		return extractor.FixedPointExtractor.ExtractFixedPoint(extractor, expression)
	}
	return extractor.ExtractFixedPoint(expression)
}

func (extractor *ExpressionExtractor) ExtractFixedPoint(expression *FixedPointExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitStringExpression(expression *StringExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.StringExtractor != nil {
		return extractor.StringExtractor.ExtractString(extractor, expression)
	}
	return extractor.ExtractString(expression)
}

func (extractor *ExpressionExtractor) ExtractString(expression *StringExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitStringTemplateExpression(expression *StringTemplateExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.StringTemplateExtractor != nil {
		return extractor.StringTemplateExtractor.ExtractStringTemplate(extractor, expression)
	}
	return extractor.ExtractStringTemplate(expression)
}

func (extractor *ExpressionExtractor) ExtractStringTemplate(expression *StringTemplateExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite all value expressions

	rewrittenExpressions, extractedExpressions :=
		extractor.VisitExpressions(expression.Expressions)

	newExpression.Expressions = rewrittenExpressions

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: extractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitArrayExpression(expression *ArrayExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.ArrayExtractor != nil {
		return extractor.ArrayExtractor.ExtractArray(extractor, expression)
	}
	return extractor.ExtractArray(expression)
}

func (extractor *ExpressionExtractor) ExtractArray(expression *ArrayExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite all value expressions

	rewrittenExpressions, extractedExpressions :=
		extractor.VisitExpressions(expression.Values)

	newExpression.Values = rewrittenExpressions

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: extractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitExpressions(
	expressions []Expression,
) (
	[]Expression, []ExtractedExpression,
) {
	var rewrittenExpressions []Expression
	var extractedExpressions []ExtractedExpression

	for _, expression := range expressions {
		result := extractor.Extract(expression)

		rewrittenExpressions = append(
			rewrittenExpressions,
			result.RewrittenExpression,
		)

		extractedExpressions = append(
			extractedExpressions,
			result.ExtractedExpressions...,
		)
	}

	return rewrittenExpressions, extractedExpressions
}

func (extractor *ExpressionExtractor) VisitDictionaryExpression(expression *DictionaryExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.DictionaryExtractor != nil {
		return extractor.DictionaryExtractor.ExtractDictionary(extractor, expression)
	}
	return extractor.ExtractDictionary(expression)
}

func (extractor *ExpressionExtractor) ExtractDictionary(expression *DictionaryExpression) ExpressionExtraction {

	var extractedExpressions []ExtractedExpression

	// copy the expression
	newExpression := *expression

	// rewrite all value expressions

	rewrittenEntries := make([]DictionaryEntry, len(expression.Entries))

	for i, entry := range expression.Entries {
		keyResult := extractor.Extract(entry.Key)
		extractedExpressions = append(extractedExpressions, keyResult.ExtractedExpressions...)

		valueResult := extractor.Extract(entry.Value)
		extractedExpressions = append(extractedExpressions, valueResult.ExtractedExpressions...)

		rewrittenEntries[i] = NewDictionaryEntry(
			extractor.MemoryGauge,
			keyResult.RewrittenExpression,
			valueResult.RewrittenExpression,
		)
	}

	newExpression.Entries = rewrittenEntries

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: extractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitIdentifierExpression(expression *IdentifierExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.IdentifierExtractor != nil {
		return extractor.IdentifierExtractor.ExtractIdentifier(extractor, expression)
	}
	return extractor.ExtractIdentifier(expression)
}

func (extractor *ExpressionExtractor) ExtractIdentifier(expression *IdentifierExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitInvocationExpression(expression *InvocationExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.InvocationExtractor != nil {
		return extractor.InvocationExtractor.ExtractInvocation(extractor, expression)
	}
	return extractor.ExtractInvocation(expression)
}

func (extractor *ExpressionExtractor) ExtractInvocation(expression *InvocationExpression) ExpressionExtraction {
	var extractedExpressions []ExtractedExpression

	invokedExpression := expression.InvokedExpression

	// copy the expression
	newExpression := *expression

	// rewrite invoked expression

	invokedExpressionResult := extractor.Extract(invokedExpression)
	newExpression.InvokedExpression = invokedExpressionResult.RewrittenExpression
	extractedExpressions = append(
		extractedExpressions,
		invokedExpressionResult.ExtractedExpressions...,
	)

	// rewrite all arguments

	newArguments, argumentExtractedExpressions := extractor.extractArguments(expression.Arguments)
	extractedExpressions = append(
		extractedExpressions,
		argumentExtractedExpressions...,
	)

	newExpression.Arguments = newArguments

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: extractedExpressions,
	}
}

func (extractor *ExpressionExtractor) extractArguments(
	arguments []*Argument,
) (
	newArguments []*Argument,
	extractedExpressions []ExtractedExpression,
) {
	for _, argument := range arguments {

		// copy the argument
		newArgument := *argument

		argumentResult := extractor.Extract(argument.Expression)

		newArgument.Expression = argumentResult.RewrittenExpression

		extractedExpressions = append(
			extractedExpressions,
			argumentResult.ExtractedExpressions...,
		)

		newArguments = append(newArguments, &newArgument)
	}
	return newArguments, extractedExpressions
}

func (extractor *ExpressionExtractor) VisitMemberExpression(expression *MemberExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.MemberExtractor != nil {
		return extractor.MemberExtractor.ExtractMember(extractor, expression)
	}
	return extractor.ExtractMember(expression)
}

func (extractor *ExpressionExtractor) ExtractMember(expression *MemberExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.Expression)

	newExpression.Expression = result.RewrittenExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitIndexExpression(expression *IndexExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.IndexExtractor != nil {
		return extractor.IndexExtractor.ExtractIndex(extractor, expression)
	}
	return extractor.ExtractIndex(expression)
}

func (extractor *ExpressionExtractor) ExtractIndex(expression *IndexExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.TargetExpression)

	newExpression.TargetExpression = result.RewrittenExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitConditionalExpression(expression *ConditionalExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.ConditionalExtractor != nil {
		return extractor.ConditionalExtractor.ExtractConditional(extractor, expression)
	}
	return extractor.ExtractConditional(expression)
}

func (extractor *ExpressionExtractor) ExtractConditional(expression *ConditionalExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite all sub-expressions

	rewrittenExpressions, extractedExpressions :=
		extractor.VisitExpressions([]Expression{
			newExpression.Test,
			newExpression.Then,
			newExpression.Else,
		})

	newExpression.Test = rewrittenExpressions[0]
	newExpression.Then = rewrittenExpressions[1]
	newExpression.Else = rewrittenExpressions[2]

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: extractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitUnaryExpression(expression *UnaryExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.UnaryExtractor != nil {
		return extractor.UnaryExtractor.ExtractUnary(extractor, expression)
	}
	return extractor.ExtractUnary(expression)
}

func (extractor *ExpressionExtractor) ExtractUnary(expression *UnaryExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.Expression)

	newExpression.Expression = result.RewrittenExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitBinaryExpression(expression *BinaryExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.BinaryExtractor != nil {
		return extractor.BinaryExtractor.ExtractBinary(extractor, expression)
	}
	return extractor.ExtractBinary(expression)
}

func (extractor *ExpressionExtractor) ExtractBinary(expression *BinaryExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite left and right sub-expression

	rewrittenExpressions, extractedExpressions :=
		extractor.VisitExpressions([]Expression{
			newExpression.Left,
			newExpression.Right,
		})

	newExpression.Left = rewrittenExpressions[0]
	newExpression.Right = rewrittenExpressions[1]

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: extractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitFunctionExpression(expression *FunctionExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.FunctionExtractor != nil {
		return extractor.FunctionExtractor.ExtractFunction(extractor, expression)
	}
	return extractor.ExtractFunction(expression)
}

func (extractor *ExpressionExtractor) ExtractFunction(_ *FunctionExpression) ExpressionExtraction {
	// NOTE: not supported
	panic(errors.NewUnreachableError())
}

func (extractor *ExpressionExtractor) VisitCastingExpression(expression *CastingExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.CastingExtractor != nil {
		return extractor.CastingExtractor.ExtractCast(extractor, expression)
	}
	return extractor.ExtractCast(expression)

}

func (extractor *ExpressionExtractor) ExtractCast(expression *CastingExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.Expression)

	newExpression.Expression = result.RewrittenExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitCreateExpression(expression *CreateExpression) ExpressionExtraction {
	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.CreateExtractor != nil {
		return extractor.CreateExtractor.ExtractCreate(extractor, expression)
	}
	return extractor.ExtractCreate(expression)
}

func (extractor *ExpressionExtractor) ExtractCreate(expression *CreateExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.InvocationExpression)

	invocationExpression, ok := result.RewrittenExpression.(*InvocationExpression)
	if !ok {
		// Edge-case:
		// The rewritten expression returned from the extractor may not be an InvocationExpression,
		// but an expression of another type.
		//
		// Wrap the rewritten expression in an InvocationExpression.

		invocationExpression = &InvocationExpression{
			InvokedExpression: result.RewrittenExpression,
			EndPos:            result.RewrittenExpression.EndPosition(extractor.MemoryGauge),
		}
	}

	newExpression.InvocationExpression = invocationExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitDestroyExpression(expression *DestroyExpression) ExpressionExtraction {
	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.DestroyExtractor != nil {
		return extractor.DestroyExtractor.ExtractDestroy(extractor, expression)
	}
	return extractor.ExtractDestroy(expression)
}

func (extractor *ExpressionExtractor) ExtractDestroy(expression *DestroyExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.Expression)

	newExpression.Expression = result.RewrittenExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitReferenceExpression(expression *ReferenceExpression) ExpressionExtraction {
	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.ReferenceExtractor != nil {
		return extractor.ReferenceExtractor.ExtractReference(extractor, expression)
	}
	return extractor.ExtractReference(expression)
}

func (extractor *ExpressionExtractor) ExtractReference(expression *ReferenceExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.Expression)

	newExpression.Expression = result.RewrittenExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitForceExpression(expression *ForceExpression) ExpressionExtraction {
	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.ForceExtractor != nil {
		return extractor.ForceExtractor.ExtractForce(extractor, expression)
	}
	return extractor.ExtractForce(expression)
}

func (extractor *ExpressionExtractor) ExtractForce(expression *ForceExpression) ExpressionExtraction {

	// copy the expression
	newExpression := *expression

	// rewrite the sub-expression

	result := extractor.Extract(newExpression.Expression)

	newExpression.Expression = result.RewrittenExpression

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: result.ExtractedExpressions,
	}
}

func (extractor *ExpressionExtractor) VisitPathExpression(expression *PathExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.PathExtractor != nil {
		return extractor.PathExtractor.ExtractPath(extractor, expression)
	}
	return extractor.ExtractPath(expression)
}

func (extractor *ExpressionExtractor) ExtractPath(expression *PathExpression) ExpressionExtraction {
	return rewriteExpressionAsIs(expression)
}

func (extractor *ExpressionExtractor) VisitAttachExpression(expression *AttachExpression) ExpressionExtraction {

	// delegate to child extractor, if any,
	// or call default implementation

	if extractor.AttachExtractor != nil {
		return extractor.AttachExtractor.ExtractAttach(extractor, expression)
	}
	return extractor.ExtractAttach(expression)
}

func (extractor *ExpressionExtractor) ExtractAttach(expression *AttachExpression) ExpressionExtraction {
	// copy the expression
	newExpression := *expression

	// rewrite left and right sub-expression

	rewrittenExpressions, extractedExpressions :=
		extractor.VisitExpressions([]Expression{
			newExpression.Attachment,
			newExpression.Base,
		})

	newExpression.Attachment = rewrittenExpressions[0].(*InvocationExpression)
	newExpression.Base = rewrittenExpressions[1]

	return ExpressionExtraction{
		RewrittenExpression:  &newExpression,
		ExtractedExpressions: extractedExpressions,
	}
}
