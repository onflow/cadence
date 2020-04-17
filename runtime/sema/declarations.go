package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type ValueDeclaration interface {
	ValueDeclarationType() Type
	ValueDeclarationKind() common.DeclarationKind
	ValueDeclarationPosition() ast.Position
	ValueDeclarationIsConstant() bool
	ValueDeclarationArgumentLabels() []string
}

type TypeDeclaration interface {
	TypeDeclarationType() Type
	TypeDeclarationKind() common.DeclarationKind
	TypeDeclarationPosition() ast.Position
}
