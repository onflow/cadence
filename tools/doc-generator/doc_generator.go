package doc_generator

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
)

var templateFiles = []string{
	"base-template",
	"declarations-template",
	"function-template",
	"composite-template",
	"field-template",
	"enum-template",
	"enum-case-template",
	"composite-full-template",
}

var functions = template.FuncMap{
	"isFunction": func(declaration ast.Declaration) bool {
		return declaration.DeclarationKind() == common.DeclarationKindFunction
	},

	"isComposite": func(declaration ast.Declaration) bool {
		switch declaration.DeclarationKind() {
		case common.DeclarationKindStructure,
			common.DeclarationKindStructureInterface,
			common.DeclarationKindResource,
			common.DeclarationKindResourceInterface,
			common.DeclarationKindContract:
			return true
		default:
			return false
		}
	},

	"isEnum": func(declaration ast.Declaration) bool {
		return declaration.DeclarationKind() == common.DeclarationKindEnum
	},

	"isInterface": func(declaration ast.Declaration) bool {
		switch declaration.DeclarationKind() {
		case common.DeclarationKindStructureInterface,
			common.DeclarationKindResourceInterface:
			return true
		default:
			return false
		}
	},

	"declKeyword": func(declaration ast.Declaration) string {
		return declaration.DeclarationKind().Keywords()
	},
}

var ROOT_DIR = func() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return filepath.Dir(wd)
}()

type DocGenerator struct {
	entryPageGen     *template.Template
	compositePageGen *template.Template
	typeNames        []string
	outputDir        string
}

func NewDocGenerator() *DocGenerator {
	gen := &DocGenerator{}

	functions["fileName"] = func(decl ast.Declaration) string {
		pathPrefix := gen.fileName()
		if len(pathPrefix) == 0 {
			return decl.DeclarationIdentifier().String() + ".md"
		}

		return gen.fileName() + "_" + decl.DeclarationIdentifier().String() + ".md"
	}

	entryPageGen := template.New("base-template")
	entryPageGen.Funcs(functions)
	entryPageGen = registerTemplates(entryPageGen)

	compositePageGen := template.New("composite-full-template")
	compositePageGen.Funcs(functions)
	compositePageGen = registerTemplates(compositePageGen)

	gen.compositePageGen = compositePageGen
	gen.entryPageGen = entryPageGen

	return gen
}

func registerTemplates(tmpl *template.Template) *template.Template {
	for _, templateFile := range templateFiles {
		var err error
		tmpl, err = tmpl.ParseFiles(path.Join(ROOT_DIR, "templates", templateFile))
		if err != nil {
			panic(err)
		}
	}

	return tmpl
}

func (gen *DocGenerator) Generate(source string, outputDir string) {
	gen.outputDir = outputDir

	program, err := parser2.ParseProgram(source)
	if err != nil {
		panic(err)
	}

	program.Accept(gen)
}

var _ ast.Visitor = NewDocGenerator()

func (gen *DocGenerator) VisitProgram(program *ast.Program) ast.Repr {

	var soleDecl ast.Declaration = program.SoleContractDeclaration()
	if soleDecl == nil {
		soleDecl = program.SoleContractInterfaceDeclaration()
	}

	// If its not a sole-declaration, and has multiple top level declarations
	// then generated an entry page.
	if soleDecl == nil {
		// Generate entry page
		f, err := os.Create(gen.outputDir + "/index.md")
		if err != nil {
			panic(err)
		}

		err = gen.entryPageGen.Execute(f, program)
		if err != nil {
			panic(err)
		}
	}

	// Generate dedicated pages for all the nested composite declarations
	for _, decl := range program.Declarations() {
		decl.Accept(gen)

	}

	return nil
}

func (gen *DocGenerator) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) ast.Repr {
	return nil
}
func (gen *DocGenerator) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) ast.Repr {
	if declaration.DeclarationKind() == common.DeclarationKindEvent {
		return nil
	}

	declName := declaration.DeclarationIdentifier().String()
	return gen.genCompositeDecl(declName, declaration.Members, declaration)
}

func (gen *DocGenerator) genCompositeDecl(name string, members *ast.Members, decl ast.Declaration) ast.Repr {
	gen.typeNames = append(gen.typeNames, name)

	defer func() {
		gen.typeNames = gen.typeNames[:len(gen.typeNames)-1]
	}()

	output := gen.outputDir + "/" + gen.fileName() + ".md"
	f, err := os.Create(output)
	if err != nil {
		panic(err)
	}

	err = gen.compositePageGen.Execute(f, decl)
	if err != nil {
		panic(err)
	}

	for _, decl := range members.Declarations() {
		decl.Accept(gen)
	}

	return nil
}

func (gen *DocGenerator) fileName() string {
	builder := strings.Builder{}
	for index, name := range gen.typeNames {
		if index > 0 {
			builder.WriteString("_")
		}
		builder.WriteString(name)
	}

	return builder.String()
}

func (gen *DocGenerator) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) ast.Repr {
	declName := declaration.DeclarationIdentifier().String()
	return gen.genCompositeDecl(declName, declaration.Members, declaration)
}

func (gen *DocGenerator) VisitFieldDeclaration(declaration *ast.FieldDeclaration) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitEnumCaseDeclaration(declaration *ast.EnumCaseDeclaration) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitPragmaDeclaration(declaration *ast.PragmaDeclaration) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitImportDeclaration(declaration *ast.ImportDeclaration) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitTransactionDeclaration(declaration *ast.TransactionDeclaration) ast.Repr {
	return nil
}

// Unused methods

func (gen *DocGenerator) VisitReturnStatement(statement *ast.ReturnStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitBreakStatement(statement *ast.BreakStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitContinueStatement(statement *ast.ContinueStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitIfStatement(statement *ast.IfStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitSwitchStatement(statement *ast.SwitchStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitWhileStatement(statement *ast.WhileStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitForStatement(statement *ast.ForStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitEmitStatement(statement *ast.EmitStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitAssignmentStatement(statement *ast.AssignmentStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitSwapStatement(statement *ast.SwapStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitExpressionStatement(statement *ast.ExpressionStatement) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitBoolExpression(expression *ast.BoolExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitNilExpression(expression *ast.NilExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitIntegerExpression(expression *ast.IntegerExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitFixedPointExpression(expression *ast.FixedPointExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitArrayExpression(expression *ast.ArrayExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitDictionaryExpression(expression *ast.DictionaryExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitIdentifierExpression(expression *ast.IdentifierExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitInvocationExpression(expression *ast.InvocationExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitMemberExpression(expression *ast.MemberExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitIndexExpression(expression *ast.IndexExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitConditionalExpression(expression *ast.ConditionalExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitUnaryExpression(expression *ast.UnaryExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitBinaryExpression(expression *ast.BinaryExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitFunctionExpression(expression *ast.FunctionExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitStringExpression(expression *ast.StringExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitCastingExpression(expression *ast.CastingExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitCreateExpression(expression *ast.CreateExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitDestroyExpression(expression *ast.DestroyExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitReferenceExpression(expression *ast.ReferenceExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitForceExpression(expression *ast.ForceExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitPathExpression(expression *ast.PathExpression) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitBlock(block *ast.Block) ast.Repr {
	return nil
}

func (gen *DocGenerator) VisitFunctionBlock(block *ast.FunctionBlock) ast.Repr {
	return nil
}
