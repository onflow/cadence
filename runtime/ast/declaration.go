package ast

import "github.com/onflow/cadence/runtime/common"

type Declaration interface {
	Element
	isDeclaration()
	DeclarationIdentifier() *Identifier
	DeclarationKind() common.DeclarationKind
	DeclarationAccess() Access
}
