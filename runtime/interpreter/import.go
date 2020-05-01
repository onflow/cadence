package interpreter

import (
	"github.com/onflow/cadence/runtime/ast"
)

// Import

type Import interface {
	isImport()
}

// VirtualImport

type VirtualImport struct {
	Globals   map[string]Value
	TypeCodes TypeCodes
}

func (VirtualImport) isImport() {}

// ProgramImport

type ProgramImport struct {
	Program *ast.Program
}

func (ProgramImport) isImport() {}
