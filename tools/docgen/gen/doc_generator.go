package gen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"text/template"

	"github.com/markbates/pkger"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
)

const nameSeparator = "_"
const mdFileExt = ".md"

const baseTemplate = "base-template"
const compositeFullTemplate = "composite-full-template"

var templateFiles = []string{
	baseTemplate,
	compositeFullTemplate,
	"declarations-template",
	"function-template",
	"composite-template",
	"field-template",
	"enum-template",
	"enum-case-template",
	"initializer-template",
}

type DocGenerator struct {
	entryPageGen     *template.Template
	compositePageGen *template.Template
	typeNames        []string
	outputDir        string
	files            InMemoryFS
}

type InMemoryFS map[string][]byte

type InMemoryWriter struct {
	fileName string
	buf      *bytes.Buffer
	fs       InMemoryFS
}

func NewInMemoryWriter(fs InMemoryFS, fileName string) *InMemoryWriter {
	return &InMemoryWriter{
		fileName: fileName,
		fs:       fs,
		buf:      &bytes.Buffer{},
	}
}

func (w *InMemoryWriter) Write(bytes []byte) (n int, err error) {
	return w.buf.Write(bytes)
}

func (w *InMemoryWriter) Close() error {
	w.fs[w.fileName] = w.buf.Bytes()
	w.buf = nil
	return nil
}

func NewDocGenerator() *DocGenerator {
	gen := &DocGenerator{}

	functions["fileName"] = func(decl ast.Declaration) string {
		fileNamePrefix := gen.currentFileName()
		if len(fileNamePrefix) == 0 {
			return fmt.Sprint(decl.DeclarationIdentifier().String(), mdFileExt)
		}

		return fmt.Sprint(fileNamePrefix, nameSeparator, decl.DeclarationIdentifier().String(), mdFileExt)
	}

	gen.entryPageGen = newTemplate(baseTemplate)
	gen.compositePageGen = newTemplate(compositeFullTemplate)

	return gen
}

func newTemplate(name string) *template.Template {
	tmpl := template.New(name).Funcs(functions)
	tmpl = registerTemplates(tmpl)
	return tmpl
}

func registerTemplates(tmpl *template.Template) *template.Template {
	info, err := pkger.Current()
	if err != nil {
		panic(err)
	}

	var filePaths = make([]string, len(templateFiles))

	for i, templateFile := range templateFiles {
		filePaths[i] = path.Join(info.Dir, "gen", "templates", templateFile)
	}

	tmpl, err = tmpl.ParseFiles(filePaths...)
	if err != nil {
		panic(err)
	}

	return tmpl
}

func (gen *DocGenerator) Generate(source string, outputDir string) error {
	gen.outputDir = outputDir
	gen.typeNames = make([]string, 0)

	program, err := parser2.ParseProgram(source)
	if err != nil {
		return err
	}

	return gen.genProgram(program)
}

func (gen *DocGenerator) GenerateInMemory(source string) (InMemoryFS, error) {
	gen.files = InMemoryFS{}
	gen.typeNames = make([]string, 0)

	program, err := parser2.ParseProgram(source)
	if err != nil {
		return nil, err
	}

	err = gen.genProgram(program)
	if err != nil {
		return nil, err
	}

	return gen.files, nil
}

func (gen *DocGenerator) genProgram(program *ast.Program) error {

	// If its not a sole-declaration, i.e: has multiple top level declarations,
	// then generated an entry page.
	if program.SoleContractDeclaration() == nil &&
		program.SoleContractInterfaceDeclaration() == nil {

		// Generate entry page
		// TODO: file name 'index' can conflict with struct names, resulting an overwrite.
		f, err := gen.fileWriter("index.md")
		if err != nil {
			return err
		}

		defer f.Close()

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

	fileName := fmt.Sprint(gen.currentFileName(), mdFileExt)
	f, err := gen.fileWriter(fileName)
	if err != nil {
		return err
	}

	defer f.Close()

	err = gen.compositePageGen.Execute(f, decl)
	if err != nil {
		return err
	}

	return gen.genDeclarations(members.Declarations())
}

func (gen *DocGenerator) fileWriter(fileName string) (io.WriteCloser, error) {
	if gen.files == nil {
		return os.Create(path.Join(gen.outputDir, fileName))
	}

	return NewInMemoryWriter(gen.files, fileName), nil
}

func (gen *DocGenerator) currentFileName() string {
	return strings.Join(gen.typeNames, nameSeparator)
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
