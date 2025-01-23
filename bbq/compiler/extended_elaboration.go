package compiler

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/sema"
)

type ExtendedElaboration struct {
	*sema.Elaboration

	interfaceMethodStaticCalls map[*ast.InvocationExpression]struct{}
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
