/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package docgen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"text/template"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/tools/docgen/templates"
)

const nameSeparator = "_"
const newline = "\n"
const mdFileExt = ".md"
const paramPrefix = "@param "
const returnPrefix = "@return "

const baseTemplate = "base-template"
const compositeFullTemplate = "composite-full-template"

var templateFiles = []string{
	baseTemplate,
	compositeFullTemplate,
	"composite-members-template",
	"function-template",
	"composite-template",
	"field-template",
	"enum-template",
	"enum-case-template",
	"initializer-template",
	"event-template",
}

type DocGenerator struct {
	entryPageGen     *template.Template
	compositePageGen *template.Template
	typeNames        []string
	outputDir        string
	files            InMemoryFiles
}

type InMemoryFiles map[string][]byte

type InMemoryFileWriter struct {
	fileName string
	buf      *bytes.Buffer
	files    InMemoryFiles
}

func NewInMemoryFileWriter(files InMemoryFiles, fileName string) *InMemoryFileWriter {
	return &InMemoryFileWriter{
		fileName: fileName,
		files:    files,
		buf:      &bytes.Buffer{},
	}
}

func (w *InMemoryFileWriter) Write(bytes []byte) (n int, err error) {
	return w.buf.Write(bytes)
}

func (w *InMemoryFileWriter) Close() error {
	w.files[w.fileName] = w.buf.Bytes()
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

	templateProvider := templates.NewMarkdownTemplateProvider()

	gen.entryPageGen = newTemplate(baseTemplate, templateProvider)
	gen.compositePageGen = newTemplate(compositeFullTemplate, templateProvider)

	return gen
}

func newTemplate(name string, templateProvider templates.TemplateProvider) *template.Template {
	rootTemplate := template.New(name).Funcs(functions)

	for _, templateFile := range templateFiles {
		content, err := templateProvider.Get(templateFile)
		if err != nil {
			panic(err)
		}

		var tmpl *template.Template
		if templateFile == name {
			tmpl = rootTemplate
		} else {
			tmpl = rootTemplate.New(name)
		}

		_, err = tmpl.Parse(content)
		if err != nil {
			panic(err)
		}
	}

	return rootTemplate
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

func (gen *DocGenerator) GenerateInMemory(source string) (InMemoryFiles, error) {
	gen.files = InMemoryFiles{}
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

	return NewInMemoryFileWriter(gen.files, fileName), nil
}

func (gen *DocGenerator) currentFileName() string {
	return strings.Join(gen.typeNames, nameSeparator)
}

var functions = template.FuncMap{
	"hasConformance": func(declaration ast.Declaration) bool {
		switch declaration.DeclarationKind() {
		case common.DeclarationKindStructure,
			common.DeclarationKindResource,
			common.DeclarationKindContract,
			common.DeclarationKindEnum:
			return true
		default:
			return false
		}
	},

	"isEnum": func(declaration ast.Declaration) bool {
		return declaration.DeclarationKind() == common.DeclarationKindEnum
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

	"enums": func(declarations []*ast.CompositeDeclaration) []*ast.CompositeDeclaration {
		decls := make([]*ast.CompositeDeclaration, 0)

		for _, decl := range declarations {
			if decl.DeclarationKind() == common.DeclarationKindEnum {
				decls = append(decls, decl)
			}
		}

		return decls
	},

	"structsAndResources": func(declarations []*ast.CompositeDeclaration) []*ast.CompositeDeclaration {
		decls := make([]*ast.CompositeDeclaration, 0)

		for _, decl := range declarations {
			switch decl.DeclarationKind() {
			case common.DeclarationKindStructure,
				common.DeclarationKindResource:
				decls = append(decls, decl)
			default:
			}
		}

		return decls
	},

	"events": func(declarations []*ast.CompositeDeclaration) []*ast.CompositeDeclaration {
		decls := make([]*ast.CompositeDeclaration, 0)
		for _, decl := range declarations {
			if decl.DeclarationKind() == common.DeclarationKindEvent {
				decls = append(decls, decl)
			}
		}
		return decls
	},

	"formatDoc": formatDocs,

	"formatFuncDoc": formatFunctionDocs,
}

func formatDocs(docString string) string {
	builder := strings.Builder{}

	// Trim leading and trailing empty lines
	docString = strings.TrimSpace(docString)

	lines := strings.Split(docString, newline)

	for i, line := range lines {
		formattedLine := strings.TrimSpace(line)
		if i > 0 {
			builder.WriteString(newline)
		}
		builder.WriteString(formattedLine)
	}

	return builder.String()
}

func formatFunctionDocs(docString string, genReturnType bool) string {
	builder := strings.Builder{}
	params := make([]string, 0)
	isPrevLineEmpty := false
	docLines := 0
	var returnDoc string

	// Trim leading and trailing empty lines
	docString = strings.TrimSpace(docString)

	lines := strings.Split(docString, newline)

	for _, line := range lines {
		formattedLine := strings.TrimSpace(line)

		if strings.HasPrefix(formattedLine, paramPrefix) {
			paramInfo := strings.TrimPrefix(formattedLine, paramPrefix)
			colonIndex := strings.IndexByte(paramInfo, ':')

			// If colon isn't there, cannot determine the param name.
			// Hence treat as a normal doc line.
			if colonIndex >= 0 {
				paramName := strings.TrimSpace(paramInfo[0:colonIndex])

				// If param name is empty, treat as a normal doc line.
				if len(paramName) > 0 {
					paramDoc := strings.TrimSpace(paramInfo[colonIndex+1:])

					var formattedParam string
					if len(paramDoc) > 0 {
						formattedParam = fmt.Sprintf("  - %s : _%s_", paramName, paramDoc)
					} else {
						formattedParam = fmt.Sprintf("  - %s", paramName)
					}

					params = append(params, formattedParam)
					continue
				}
			}
		} else if genReturnType && strings.HasPrefix(formattedLine, returnPrefix) {
			returnDoc = formattedLine
			continue
		}

		// Ignore the line if its a consecutive blank line.
		isLineEmpty := len(formattedLine) == 0
		if isPrevLineEmpty && isLineEmpty {
			continue
		}

		if docLines > 0 {
			builder.WriteString(newline)
		}

		builder.WriteString(formattedLine)
		isPrevLineEmpty = isLineEmpty
		docLines++
	}

	// Print the parameters
	if len(params) > 0 {
		if !isPrevLineEmpty {
			builder.WriteString(newline)
		}

		builder.WriteString(newline)
		builder.WriteString("Parameters:")

		for _, param := range params {
			builder.WriteString(newline)
			builder.WriteString(param)
		}

		isPrevLineEmpty = false
	}

	// Print the return type info
	if len(returnDoc) > 0 {
		if !isPrevLineEmpty {
			builder.WriteString(newline)
		}

		builder.WriteString(newline)

		returnInfo := strings.TrimPrefix(returnDoc, returnPrefix)
		returnInfo = strings.TrimSpace(returnInfo)
		builder.WriteString(fmt.Sprintf("Returns: %s", returnInfo))
	}

	return builder.String()
}
