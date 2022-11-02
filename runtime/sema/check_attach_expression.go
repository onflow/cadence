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
	attachment := expression.Attachment

	ty := checker.checkInvocationExpression(attachment)
	if ty.IsInvalidType() {
		return ty
	}

	attachmentType, baseIsCompositeType := ty.(*CompositeType)
	if !(baseIsCompositeType && attachmentType.Kind == common.CompositeKindAttachment) {
		checker.report(
			&AttachNonAttachmentError{
				Type:  ty,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, attachment),
			},
		)
		return InvalidType
	}

	annotatedBaseType := attachmentType.baseType
	baseExpression := expression.Base

	visibleBaseType, actualBaseType := checker.visitExpression(baseExpression, annotatedBaseType)
	if visibleBaseType.IsInvalidType() {
		return visibleBaseType
	}

	checker.checkVariableMove(baseExpression)
	checker.checkResourceMoveOperation(baseExpression, ty)

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
	// anyresource/anystruct, we need to enforce that actualBaseType is a resource or struct
	if _, annotatedIsCompositeType := annotatedBaseType.(CompositeKindedType); !annotatedIsCompositeType {
		switch baseType := actualBaseType.(type) {
		case CompositeKindedType:
			compositeKind := baseType.GetCompositeKind()
			if !(compositeKind == common.CompositeKindResource || compositeKind == common.CompositeKindStructure) {
				return reportInvalidBase(actualBaseType)
			}
		// these are always resource/structure types
		case *RestrictedType:
			break
		default:
			return reportInvalidBase(actualBaseType)
		}
	}

	return visibleBaseType
}
