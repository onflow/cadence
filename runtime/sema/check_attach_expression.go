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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitAttachExpression(expression *ast.AttachExpression) Type {

	if !checker.Config.AttachmentsEnabled {
		checker.report(&AttachmentsNotEnabledError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
		})
	}

	attachment := expression.Attachment
	baseExpression := expression.Base

	baseType := checker.VisitExpression(baseExpression, checker.expectedType)
	attachmentType := checker.checkInvocationExpression(attachment)

	if attachmentType.IsInvalidType() || baseType.IsInvalidType() {
		return InvalidType
	}

	checker.checkVariableMove(baseExpression)
	checker.checkResourceMoveOperation(baseExpression, attachmentType)

	// check that the attachment type is a valid attachment,
	// and that it is a subtype of the declared base type
	attachmentCompositeType, isCompositeType := attachmentType.(*CompositeType)
	if !(isCompositeType && attachmentCompositeType.Kind == common.CompositeKindAttachment) {
		checker.report(
			&AttachNonAttachmentError{
				Type:  attachmentType,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, attachment),
			},
		)
		return InvalidType
	}

	declaredBaseType := attachmentCompositeType.baseType

	if !IsSubType(baseType, declaredBaseType) {
		checker.report(
			&TypeMismatchError{
				ExpectedType: declaredBaseType,
				ActualType:   baseType,
				Expression:   baseExpression,
				Range:        checker.expressionRange(baseExpression),
			},
		)
		return InvalidType
	}

	reportInvalidBase := func(ty Type) *SimpleType {
		checker.report(
			&AttachToInvalidTypeError{
				Type:  ty,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, attachment),
			},
		)
		return InvalidType
	}

	// if the annotatedBaseType is a specific interface or composite, then the above code will already have
	// checked that the type of the base expression is also a composite. However, if the annotatedBaseType is
	// anyresource/anystruct, we need to enforce that baseType is a resource or struct, to prevent
	// permitting code like `attach A to 4`, if `A` was declared for `AnyStruct`.
	//
	// Note that we cannot just check
	// that the base type is type-indexable, because it could be a reference type, which is indexable for
	// attachment-access purposes, but is not a valid attach target.
	if _, annotatedIsCompositeType := declaredBaseType.(CompositeKindedType); !annotatedIsCompositeType {
		switch baseType := baseType.(type) {
		case CompositeKindedType:
			compositeKind := baseType.GetCompositeKind()
			if !compositeKind.SupportsAttachments() {
				return reportInvalidBase(baseType)
			}
		// these are always resource/structure types
		case *IntersectionType:
			break
		default:
			return reportInvalidBase(baseType)
		}
	}

	checker.Elaboration.SetAttachTypes(expression, attachmentCompositeType)

	return baseType
}
