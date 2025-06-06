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

package sema

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

func (checker *Checker) VisitRemoveStatement(statement *ast.RemoveStatement) (_ struct{}) {

	nominalType := checker.convertNominalType(statement.Attachment)
	base := checker.VisitExpression(statement.Value, statement, nil)
	checker.checkUnusedExpressionResourceLoss(base, statement.Value)

	if nominalType == InvalidType {
		return
	}

	attachmentType, isCompositeType := nominalType.(*CompositeType)

	if !isCompositeType || attachmentType.Kind != common.CompositeKindAttachment {
		checker.report(
			&InvalidAttachmentRemoveError{
				Attachment: nominalType,
				Range:      ast.NewRangeFromPositioned(checker.memoryGauge, statement.Attachment),
			},
		)
		return
	}

	// the actual base type must be a composite that can receive an attachment,
	// and it must also be a valid subtype of the declared base type
	switch baseType := base.(type) {
	case *CompositeType:
		if !baseType.Kind.SupportsAttachments() ||
			!IsSubType(baseType, attachmentType.baseType) {
			checker.report(
				&InvalidAttachmentRemoveError{
					Attachment: nominalType,
					BaseType:   base,
					Range:      ast.NewRangeFromPositioned(checker.memoryGauge, statement),
				},
			)
		}
	case *IntersectionType:
		if !IsSubType(baseType, attachmentType.baseType) {
			checker.report(
				&InvalidAttachmentRemoveError{
					Attachment: nominalType,
					BaseType:   base,
					Range:      ast.NewRangeFromPositioned(checker.memoryGauge, statement),
				},
			)
		}
	default:
		checker.report(
			&InvalidAttachmentRemoveError{
				Attachment: nominalType,
				BaseType:   base,
				Range:      ast.NewRangeFromPositioned(checker.memoryGauge, statement),
			},
		)
	}

	checker.enforceViewAssignment(statement, statement.Value)

	checker.Elaboration.SetAttachmentRemoveTypes(statement, nominalType)

	return
}
