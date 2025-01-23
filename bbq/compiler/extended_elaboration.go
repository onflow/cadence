package compiler

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/sema"
)

type ExtendedElaboration struct {
	*sema.Elaboration

	interfaceMethodStaticCalls map[*ast.InvocationExpression]struct{}
	interfaceDeclarationTypes  map[*ast.InterfaceDeclaration]*sema.InterfaceType
}

func NewExtendedElaboration(elaboration *sema.Elaboration) *ExtendedElaboration {
	return &ExtendedElaboration{
		Elaboration: elaboration,
	}
}

func (e *ExtendedElaboration) SetInterfaceMethodStaticCall(invocation *ast.InvocationExpression) {
	if e.interfaceMethodStaticCalls == nil {
		e.interfaceMethodStaticCalls = make(map[*ast.InvocationExpression]struct{})
	}
	e.interfaceMethodStaticCalls[invocation] = struct{}{}
}

func (e *ExtendedElaboration) IsInterfaceMethodStaticCall(invocation *ast.InvocationExpression) bool {
	if e.interfaceMethodStaticCalls == nil {
		return false
	}

	_, ok := e.interfaceMethodStaticCalls[invocation]
	return ok
}

func (e *ExtendedElaboration) SetInterfaceDeclarationType(decl *ast.InterfaceDeclaration, interfaceType *sema.InterfaceType) {
	if e.interfaceDeclarationTypes == nil {
		e.interfaceDeclarationTypes = make(map[*ast.InterfaceDeclaration]*sema.InterfaceType)
	}

	e.interfaceDeclarationTypes[decl] = interfaceType
}

func (e *ExtendedElaboration) InterfaceDeclarationType(decl *ast.InterfaceDeclaration) *sema.InterfaceType {
	// First lookup in the extended type info
	typ, ok := e.interfaceDeclarationTypes[decl]
	if ok {
		return typ
	}

	// If couldn't find, then fallback and look in the original elaboration
	return e.Elaboration.InterfaceDeclarationType(decl)
}
