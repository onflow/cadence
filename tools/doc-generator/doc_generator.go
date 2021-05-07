package doc_generator

import (
	"log"
	"os"
	"text/template"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
)

var templateFiles = []string{
	"templates/base-template",
	"templates/declaration-template",
	"templates/function-template",
	"templates/struct-template",
	"templates/field-template",
	"templates/enum-template",
	"templates/enum-case-template",
}

var functions = template.FuncMap{
	"isFunction": func(declaration ast.Declaration) bool {
		return declaration.DeclarationKind() == common.DeclarationKindFunction
	},

	"isStruct": func(declaration ast.Declaration) bool {
		return declaration.DeclarationKind() == common.DeclarationKindStructure
	},

	"isEnum": func(declaration ast.Declaration) bool {
		return declaration.DeclarationKind() == common.DeclarationKindEnum
	},
}

type DocGenerator struct {
	tmpl *template.Template
}

func NewDocGenerator() *DocGenerator {
	tmpl := template.New("base-template")

	// Register functions
	tmpl.Funcs(functions)

	// Register template files
	for _, templateFile := range templateFiles {
		var err error
		tmpl, err = tmpl.ParseFiles(templateFile)
		if err != nil {
			panic(err)
		}
	}

	return &DocGenerator{
		tmpl: tmpl,
	}
}

func (gen *DocGenerator) generate(source string, outputPath string) {
	program, err := parser2.ParseProgram(source)
	if err != nil {
		panic(err)
	}

	// Output writer
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}

	// Generate docs
	err = gen.tmpl.Execute(f, program)
	if err != nil {
		panic(err)
	}
}
