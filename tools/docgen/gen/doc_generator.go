package gen

import (
	"os"
	"path"
	"strings"

	"text/template"

	"github.com/markbates/pkger"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
)

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
		info, err := pkger.Current()
		if err != nil {
			panic(err)
		}

		filePath := path.Join(info.Dir, "gen", "templates", templateFile)

		tmpl, err = tmpl.ParseFiles(filePath)
		if err != nil {
			panic(err)
		}
	}

	return tmpl
}

func (gen *DocGenerator) Generate(source string, outputDir string) error {
	gen.outputDir = outputDir

	program, err := parser2.ParseProgram(source)
	if err != nil {
		return err
	}

	return gen.genProgram(program)
}

func (gen *DocGenerator) genProgram(program *ast.Program) error {

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
			return err
		}

		err = gen.entryPageGen.Execute(f, program)
		if err != nil {
			return err
		}
	}

	// Generate dedicated pages for all the nested composite declarations
	return gen.genDeclarations(program.Declarations())
}

func (gen *DocGenerator) genDeclarations(decls []ast.Declaration) error {
	var err error
	for _, decl := range decls {
		switch astDecl := decl.(type) {
		case *ast.CompositeDeclaration:
			err = gen.genCompositeDeclaration(astDecl)
		case *ast.InterfaceDeclaration:
			err = gen.genInterfaceDeclaration(astDecl)
		default:
			// do nothing
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (gen *DocGenerator) genCompositeDeclaration(declaration *ast.CompositeDeclaration) error {
	if declaration.DeclarationKind() == common.DeclarationKindEvent {
		return nil
	}

	declName := declaration.DeclarationIdentifier().String()
	return gen.genCompositeDecl(declName, declaration.Members, declaration)
}

func (gen *DocGenerator) genInterfaceDeclaration(declaration *ast.InterfaceDeclaration) error {
	declName := declaration.DeclarationIdentifier().String()
	return gen.genCompositeDecl(declName, declaration.Members, declaration)
}

func (gen *DocGenerator) genCompositeDecl(name string, members *ast.Members, decl ast.Declaration) error {
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

	return gen.genDeclarations(members.Declarations())
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

var templateFiles = []string{
	"base-template",
	"declarations-template",
	"function-template",
	"composite-template",
	"field-template",
	"enum-template",
	"enum-case-template",
	"composite-full-template",
	"initializer-template",
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

	"declTypeTitle": func(declaration ast.Declaration) string {
		return strings.Title(declaration.DeclarationKind().Keywords())
	},

	"genInitializer": func(declaration ast.Declaration) bool {
		switch declaration.DeclarationKind() {
		case common.DeclarationKindStructure,
			common.DeclarationKindResource:
			return true
		default:
			return false
		}
	},
}
