package compiler

import (
	"sync"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type DesugaredProgram struct {
	*ast.Program

	once                   sync.Once
	_functionDeclarations  []ast.Declaration
	_compositeDeclarations []ast.CompositeLikeDeclaration
}

func NewDesugaredProgram(memoryGauge common.MemoryGauge, program *ast.Program) *DesugaredProgram {
	common.UseMemory(memoryGauge, common.ProgramMemoryUsage)
	return &DesugaredProgram{
		Program: program,
	}
}

func (p *DesugaredProgram) FunctionDeclarations() []ast.Declaration {
	p.once.Do(p.initializer())
	return p._functionDeclarations
}

func (p *DesugaredProgram) CompositeDeclarations() []ast.CompositeLikeDeclaration {
	p.once.Do(p.initializer())
	return p._compositeDeclarations
}

func (p *DesugaredProgram) initializer() func() {
	return func() {
		p.init()
	}
}

func (p *DesugaredProgram) init() {

	// Important: allocate instead of nil
	p._functionDeclarations = make([]ast.Declaration, 0)

	for _, declaration := range p.Declarations() {

		switch declaration := declaration.(type) {
		case *ast.FunctionDeclaration, *DesugaredFunctionDeclaration:
			p._functionDeclarations = append(p._functionDeclarations, declaration)
		case *ast.CompositeDeclaration:
			p._compositeDeclarations = append(p._compositeDeclarations, declaration)
		case *DesugaredCompositeDeclaration:
			p._compositeDeclarations = append(p._compositeDeclarations, declaration)
		}
	}
}

type DesugaredMembers struct {
	*ast.Members

	once                   sync.Once
	_functionDeclarations  []ast.Declaration
	_compositeDeclarations []ast.CompositeLikeDeclaration
}

func NewDesugaredMembers(memoryGauge common.MemoryGauge, members *ast.Members) *DesugaredMembers {
	common.UseMemory(memoryGauge, common.ProgramMemoryUsage)
	return &DesugaredMembers{
		Members: members,
	}
}

func (p *DesugaredMembers) Functions() []ast.Declaration {
	p.once.Do(p.initializer())
	return p._functionDeclarations
}

func (p *DesugaredMembers) Composites() []ast.CompositeLikeDeclaration {
	p.once.Do(p.initializer())
	return p._compositeDeclarations
}

func (p *DesugaredMembers) initializer() func() {
	return func() {
		p.init()
	}
}

func (p *DesugaredMembers) init() {

	// Important: allocate instead of nil
	p._functionDeclarations = make([]ast.Declaration, 0)

	for _, declaration := range p.Declarations() {

		switch declaration := declaration.(type) {
		case *ast.FunctionDeclaration, *DesugaredFunctionDeclaration:
			p._functionDeclarations = append(p._functionDeclarations, declaration)
		case *ast.CompositeDeclaration:
			p._compositeDeclarations = append(p._compositeDeclarations, declaration)
		case *DesugaredCompositeDeclaration:
			p._compositeDeclarations = append(p._compositeDeclarations, declaration)
		}
	}
}

type DesugaredFunctionDeclaration struct {
	*sema.Elaboration
	*ast.FunctionDeclaration
}

func NewDesugaredFunctionDeclaration(
	elaboration *sema.Elaboration,
	declaration *ast.FunctionDeclaration,
) *DesugaredFunctionDeclaration {
	return &DesugaredFunctionDeclaration{
		Elaboration:         elaboration,
		FunctionDeclaration: declaration,
	}
}

type DesugaredCompositeDeclaration struct {
	*ast.CompositeDeclaration
	*DesugaredMembers
}

func NewDesugaredCompositeDeclaration(
	declaration *ast.CompositeDeclaration,
	members *DesugaredMembers,
) *DesugaredCompositeDeclaration {
	return &DesugaredCompositeDeclaration{
		CompositeDeclaration: declaration,
		DesugaredMembers:     members,
	}
}

type ExtendedDeclarationVisitor[T any] interface {
	ast.DeclarationVisitor[T]
	VisitDesugaredFunctionDeclaration(*DesugaredFunctionDeclaration) T
}
