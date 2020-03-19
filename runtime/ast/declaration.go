package ast

import "github.com/dapperlabs/cadence/runtime/common"

type Declaration interface {
	Element
	isDeclaration()
	DeclarationIdentifier() *Identifier
	DeclarationKind() common.DeclarationKind
	DeclarationAccess() Access
}
