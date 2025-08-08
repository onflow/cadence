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

package main

import (
	"fmt"
	"math/big"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/parser/lexer"
	"github.com/onflow/cadence/sema"
)

var placeholderString = "placeholder"

const placeholderInt = 42

var placeholderStrings = []string{placeholderString}

var placeholderIntegerLiteralKind = common.IntegerLiteralKindDecimal

var placeholderInvalidNumberLiteralKind = parser.InvalidNumberLiteralKindUnknownPrefix

var placeholderPosition = ast.Position{Offset: 1, Line: 2, Column: 3}

var placeholderEndPosition = ast.Position{Offset: 4, Line: 5, Column: 6}

var placeholderRange = ast.Range{
	StartPos: placeholderPosition,
	EndPos:   placeholderEndPosition,
}

const placeholderTypeName = "PlaceholderType"

var placeholderLocation = common.AddressLocation{
	Name:    placeholderTypeName,
	Address: common.MustBytesToAddress([]byte{0x1}),
}

var placeholderCompositeKind = common.CompositeKindResource

var placeholderSemaType = sema.AnyType

var placeholderCompositeLikeDeclaration = &ast.CompositeDeclaration{
	Identifier: ast.Identifier{
		Identifier: placeholderTypeName,
		Pos:        placeholderPosition,
	},
	Access:        placeholderAstAccess,
	CompositeKind: common.CompositeKindResource,
}

var placeholderAstAccess = ast.AccessAll

var placeholderDeclarationKind = common.DeclarationKindResource

var placeholderCompositeType = &sema.CompositeType{
	Location:   placeholderLocation,
	Identifier: placeholderTypeName,
	Kind:       common.CompositeKindResource,
}

var placeholderInterfaceType = &sema.InterfaceType{
	Location:      placeholderLocation,
	Identifier:    placeholderTypeName,
	CompositeKind: common.CompositeKindResource,
}

var placeholderTransferOperation = ast.TransferOperationMove

var placeholderInitializerMismatch = &sema.InitializerMismatch{
	CompositePurity:     placeholderPurity,
	InterfacePurity:     placeholderPurity,
	CompositeParameters: placeholderParameters,
	InterfaceParameters: placeholderParameters,
}

var placeholderPurity = sema.FunctionPurityView

var placeholderParameters = []sema.Parameter{
	{
		Identifier:     "placeholderParameter",
		TypeAnnotation: sema.NewTypeAnnotation(sema.AnyType),
	},
}

var placeholderEntitlementMapType = &sema.EntitlementMapType{
	Location:   placeholderLocation,
	Identifier: placeholderTypeName,
}

var placeholderEntitlementType = &sema.EntitlementType{
	Location:   placeholderLocation,
	Identifier: placeholderTypeName,
}

var placeholderSemaAccess = sema.PrimitiveAccess(ast.AccessAll)

var placeholderError = fmt.Errorf("placeholder error") //nolint:staticcheck

var placeholderOperandSide = common.OperandSideLeft

var placeholderBigInt = big.NewInt(42)

var placeholderOperation = ast.OperationAnd

var placeholderNominalType = &ast.NominalType{
	Identifier: ast.Identifier{
		Identifier: placeholderTypeName,
	},
}

var placeholderVariableKind = ast.VariableKindConstant

var placeholderMember = &sema.Member{
	TypeAnnotation: sema.NewTypeAnnotation(sema.AnyType),
	Identifier: ast.Identifier{
		Identifier: "placeholderField",
	},
	Access:          sema.PrimitiveAccess(ast.AccessAll),
	DeclarationKind: common.DeclarationKindField,
	VariableKind:    ast.VariableKindConstant,
}

var placeholderMembers = []*sema.Member{
	placeholderMember,
}

var placeholderMemberMismatches = []sema.MemberMismatch{
	{
		CompositeMember: &sema.Member{
			TypeAnnotation: sema.NewTypeAnnotation(sema.AnyType),
			Identifier: ast.Identifier{
				Identifier: "placeholderImplementationField",
			},
			Access:          sema.PrimitiveAccess(ast.AccessAll),
			DeclarationKind: common.DeclarationKindField,
			VariableKind:    ast.VariableKindConstant,
		},
		InterfaceMember: &sema.Member{
			TypeAnnotation: sema.NewTypeAnnotation(sema.AnyType),
			Identifier: ast.Identifier{
				Identifier: "placeholderInterfaceField",
			},
			Access:          sema.PrimitiveAccess(ast.AccessAll),
			DeclarationKind: common.DeclarationKindField,
			VariableKind:    ast.VariableKindConstant,
		},
	},
}

var placeholderCompositeTypes = []*sema.CompositeType{
	placeholderCompositeType,
}

var placeholderControlStatement = common.ControlStatementBreak

var placeholderDefaultDestroyInvalidArgumentKind = sema.InvalidExpression

var placeholderResourceInvalidationKind = sema.ResourceInvalidationKindDestroyDefinite

var placeholderResourceInvalidation = sema.ResourceInvalidation{
	Kind:     placeholderResourceInvalidationKind,
	StartPos: placeholderPosition,
	EndPos:   placeholderEndPosition,
}

var placeholderOptionalType = &sema.OptionalType{
	Type: sema.AnyType,
}

var placeholderDeclaration = placeholderCompositeLikeDeclaration

var placeholderIdentifier = &ast.Identifier{
	Identifier: "placeholder",
}

var placeholderReferenceType = &sema.ReferenceType{
	Type:          sema.IntType,
	Authorization: sema.UnauthorizedAccess,
}

var placeholderAstType = placeholderNominalType

var placeholderIdentifierExpression = &ast.IdentifierExpression{
	Identifier: ast.Identifier{
		Identifier: placeholderString,
	},
}
var placeholderMemberExpression = &ast.MemberExpression{
	Expression: placeholderIdentifierExpression,
	Identifier: ast.Identifier{
		Identifier: "placeholderField",
	},
}

var placeholderExpression = placeholderMemberExpression

var placeholderTypeParameter = &sema.TypeParameter{
	Name: placeholderTypeName,
}

var placeholderCompositeKindedType = placeholderCompositeType

var placeholderTwoSemaAccessArray = [2]sema.Access{
	sema.PrimitiveAccess(ast.AccessAll),
	sema.PrimitiveAccess(ast.AccessSelf),
}

var placeholderEntitlementSetAccess = sema.NewEntitlementSetAccess(
	[]*sema.EntitlementType{
		placeholderEntitlementType,
	},
	sema.Conjunction,
)

const placeholderTokenType = lexer.TokenIdentifier

var placeholderToken = lexer.Token{
	Type:  placeholderTokenType,
	Range: placeholderRange,
}
