package opcode

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/errors"
)

type CastKind byte

const (
	SimpleCast CastKind = iota
	FailableCast
	ForceCast
)

func CastKindFrom(operation ast.Operation) CastKind {
	switch operation {
	case ast.OperationCast:
		return SimpleCast
	case ast.OperationFailableCast:
		return FailableCast
	case ast.OperationForceCast:
		return ForceCast
	default:
		panic(errors.NewUnreachableError())
	}
}
