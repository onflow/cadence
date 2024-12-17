package main

import (
	"bytes"
	"fmt"
	"go/token"
	"io"
	"os"
	"unicode"
	"unicode/utf8"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/guess"
	"github.com/goccy/go-yaml"
)

const (
	opcodePackagePath = "github.com/onflow/cadence/bbq/opcode"
	commonPackagePath = "github.com/onflow/cadence/common"
	errorsPackagePath = "github.com/onflow/cadence/errors"
)

const headerFormat = `// Code generated by gen/main.go from %s. DO NOT EDIT.

`

const typeCommentFormat = `// %s
//
// %s
`

type operandType string

const (
	operandTypeBool          = "bool"
	operandTypeIndex         = "index"
	operandTypeIndices       = "indices"
	operandTypeSize          = "size"
	operandTypeString        = "string"
	operandTypeCastKind      = "castKind"
	operandTypePathDomain    = "pathDomain"
	operandTypeCompositeKind = "compositeKind"
)

type instruction struct {
	Name        string
	Description string
	Operands    []operand
}

type operand struct {
	Name        string
	Description string
	Type        operandType
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: gen instructions.yml instructions.go")
	}

	yamlPath := os.Args[1]
	goPath := os.Args[2]

	yamlContents, err := os.ReadFile(yamlPath)
	if err != nil {
		panic(err)
	}

	var instructions []instruction

	err = yaml.Unmarshal(yamlContents, &instructions)
	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer
	_, err = fmt.Fprintf(&buffer, headerFormat, yamlPath)
	if err != nil {
		panic(err)
	}

	var decls []dst.Decl
	for _, instruction := range instructions {
		decls = append(
			decls,
			instructionDecls(instruction)...,
		)
	}
	decls = append(decls, decodeInstructionFuncDecl(instructions))

	writeGoFile(&buffer, decls, opcodePackagePath)

	err = os.WriteFile(goPath, buffer.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}

func decodeInstructionFuncDecl(instructions []instruction) *dst.FuncDecl {

	var caseStmts []dst.Stmt

	for _, ins := range instructions {

		var resultExpr dst.Expr
		if len(ins.Operands) == 0 {
			resultExpr = &dst.CompositeLit{
				Type: instructionIdent(ins),
			}
		} else {
			resultExpr = &dst.CallExpr{
				Fun: &dst.Ident{
					Name: "Decode" + firstUpper(ins.Name),
				},
				Args: []dst.Expr{
					dst.NewIdent("ip"),
					dst.NewIdent("code"),
				},
			}
		}

		caseStmts = append(
			caseStmts,
			&dst.CaseClause{
				List: []dst.Expr{
					dst.NewIdent(firstUpper(ins.Name)),
				},
				Body: []dst.Stmt{
					&dst.ReturnStmt{
						Results: []dst.Expr{
							resultExpr,
						},
					},
				},
			},
		)
	}

	switchStmt := &dst.SwitchStmt{
		Tag: &dst.CallExpr{
			Fun: &dst.Ident{
				Name: "Opcode",
				Path: opcodePackagePath,
			},
			Args: []dst.Expr{
				&dst.CallExpr{

					Fun: &dst.Ident{
						Name: "decodeByte",
						Path: opcodePackagePath,
					},
					Args: []dst.Expr{
						dst.NewIdent("ip"),
						dst.NewIdent("code"),
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: caseStmts,
		},
		Decs: dst.SwitchStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.NewLine,
			},
		},
	}

	stmts := []dst.Stmt{
		switchStmt,
		&dst.ExprStmt{
			X: &dst.CallExpr{
				Fun: dst.NewIdent("panic"),
				Args: []dst.Expr{
					&dst.CallExpr{
						Fun: &dst.Ident{
							Name: "NewUnreachableError",
							Path: errorsPackagePath,
						},
					},
				},
			},
			Decs: dst.ExprStmtDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.EmptyLine,
					After:  dst.NewLine,
				},
			},
		},
	}

	return &dst.FuncDecl{
		Name: dst.NewIdent("DecodeInstruction"),
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{
							dst.NewIdent("ip"),
						},
						Type: &dst.StarExpr{
							X: dst.NewIdent("uint16"),
						},
					},
					{
						Names: []*dst.Ident{
							dst.NewIdent("code"),
						},
						Type: &dst.ArrayType{
							Elt: dst.NewIdent("byte"),
						},
					},
				},
			},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Type: &dst.Ident{
							Name: "Instruction",
							Path: opcodePackagePath,
						},
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}
}

func instructionDecls(instruction instruction) []dst.Decl {
	decls := []dst.Decl{
		instructionTypeDecl(instruction),
		instructionConformanceDecl(instruction),
		instructionOpcodeFuncDecl(instruction),
		instructionStringFuncDecl(instruction),
		instructionEncodeFuncDecl(instruction),
	}
	if len(instruction.Operands) > 0 {
		decls = append(decls, instructionDecodeFuncDecl(instruction))
	}
	return decls
}

func instructionIdent(ins instruction) *dst.Ident {
	return dst.NewIdent("Instruction" + firstUpper(ins.Name))
}

func operandIdent(o operand) *dst.Ident {
	return dst.NewIdent(firstUpper(o.Name))
}

func instructionTypeDecl(ins instruction) dst.Decl {
	comment := fmt.Sprintf(
		typeCommentFormat,
		instructionIdent(ins).String(),
		ins.Description,
	)

	return &dst.GenDecl{
		Tok: token.TYPE,
		Specs: []dst.Spec{
			&dst.TypeSpec{
				Name: instructionIdent(ins),
				Type: &dst.StructType{
					Fields: instructionOperandsFields(ins),
				},
			},
		},
		Decs: dst.GenDeclDecorations{
			NodeDecs: dst.NodeDecs{
				Start: []string{comment},
			},
		},
	}
}

func instructionOperandsFields(ins instruction) *dst.FieldList {
	fields := make([]*dst.Field, len(ins.Operands))
	for i, operand := range ins.Operands {

		var typeExpr dst.Expr

		switch operand.Type {
		case operandTypeBool:
			typeExpr = dst.NewIdent("bool")

		case operandTypeIndex,
			operandTypeSize:

			typeExpr = dst.NewIdent("uint16")

		case operandTypeIndices:
			typeExpr = &dst.ArrayType{
				Elt: dst.NewIdent("uint16"),
			}

		case operandTypeString:
			typeExpr = dst.NewIdent("string")

		case operandTypeCastKind:
			typeExpr = &dst.Ident{
				Name: "CastKind",
				Path: opcodePackagePath,
			}

		case operandTypePathDomain:
			typeExpr = &dst.Ident{
				Name: "PathDomain",
				Path: commonPackagePath,
			}

		case operandTypeCompositeKind:
			typeExpr = &dst.Ident{
				Name: "CompositeKind",
				Path: commonPackagePath,
			}

		default:
			panic(fmt.Sprintf("unsupported operand type: %s", operand.Type))
		}

		fields[i] = &dst.Field{
			Names: []*dst.Ident{
				operandIdent(operand),
			},
			Type: typeExpr,
		}
	}
	return &dst.FieldList{
		List: fields,
	}
}

func instructionConformanceDecl(ins instruction) *dst.GenDecl {
	return &dst.GenDecl{
		Tok: token.VAR,
		Specs: []dst.Spec{
			&dst.ValueSpec{
				Names: []*dst.Ident{
					dst.NewIdent("_"),
				},
				Type: &dst.Ident{
					Name: "Instruction",
					Path: opcodePackagePath,
				},
				Values: []dst.Expr{
					&dst.CompositeLit{
						Type: instructionIdent(ins),
					},
				},
			},
		},
	}
}

func instructionOpcodeFuncDecl(ins instruction) *dst.FuncDecl {
	stmt := &dst.ReturnStmt{
		Results: []dst.Expr{
			dst.NewIdent(firstUpper(ins.Name)),
		},
		Decs: dst.ReturnStmtDecorations{
			NodeDecs: dst.NodeDecs{
				Before: dst.NewLine,
				After:  dst.NewLine,
			},
		},
	}
	return &dst.FuncDecl{
		Recv: &dst.FieldList{
			List: []*dst.Field{
				{
					Type: instructionIdent(ins),
				},
			},
		},
		Name: dst.NewIdent("Opcode"),
		Type: &dst.FuncType{
			Params: &dst.FieldList{},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Type: &dst.Ident{
							Name: "Opcode",
							Path: opcodePackagePath,
						},
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				stmt,
			},
		},
	}
}

func instructionStringFuncDecl(ins instruction) *dst.FuncDecl {

	var stmts []dst.Stmt

	opcodeStringExpr := &dst.CallExpr{
		Fun: &dst.SelectorExpr{
			X: &dst.CallExpr{
				Fun: &dst.SelectorExpr{
					X:   dst.NewIdent("i"),
					Sel: dst.NewIdent("Opcode"),
				},
			},
			Sel: dst.NewIdent("String"),
		},
	}

	if len(ins.Operands) == 0 {
		stmt := &dst.ReturnStmt{
			Results: []dst.Expr{
				opcodeStringExpr,
			},
			Decs: dst.ReturnStmtDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.NewLine,
					After:  dst.NewLine,
				},
			},
		}
		stmts = append(stmts, stmt)
	} else {

		stmts = append(
			stmts,
			&dst.DeclStmt{
				Decl: &dst.GenDecl{
					Tok: token.VAR,
					Specs: []dst.Spec{
						&dst.ValueSpec{
							Names: []*dst.Ident{
								dst.NewIdent("sb"),
							},
							Type: &dst.Ident{
								Name: "Builder",
								Path: "strings",
							},
						},
					},
				},
				Decs: dst.DeclStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Before: dst.NewLine,
						After:  dst.NewLine,
					},
				},
			},
		)

		stmts = append(
			stmts,
			&dst.ExprStmt{
				X: &dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   dst.NewIdent("sb"),
						Sel: dst.NewIdent("WriteString"),
					},
					Args: []dst.Expr{
						opcodeStringExpr,
					},
				},
			},
		)

		for _, operand := range ins.Operands {
			var funcName string
			switch operand.Type {
			case operandTypeIndices:
				funcName = "printfUInt16ArrayArgument"
			default:
				funcName = "printfArgument"
			}
			stmts = append(
				stmts,
				&dst.ExprStmt{
					X: &dst.CallExpr{
						Fun: &dst.Ident{
							Name: funcName,
						},
						Args: []dst.Expr{
							&dst.UnaryExpr{
								Op: token.AND,
								X:  dst.NewIdent("sb"),
							},
							&dst.BasicLit{
								Kind:  token.STRING,
								Value: fmt.Sprintf(`"%s"`, operand.Name),
							},
							&dst.SelectorExpr{
								X:   dst.NewIdent("i"),
								Sel: operandIdent(operand),
							},
						},
					},
				},
			)
		}

		stmts = append(
			stmts,
			&dst.ReturnStmt{
				Results: []dst.Expr{
					&dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X:   dst.NewIdent("sb"),
							Sel: dst.NewIdent("String"),
						},
					},
				},
				Decs: dst.ReturnStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Before: dst.NewLine,
						After:  dst.NewLine,
					},
				},
			},
		)
	}

	return &dst.FuncDecl{
		Recv: &dst.FieldList{
			List: []*dst.Field{
				{
					Names: []*dst.Ident{
						dst.NewIdent("i"),
					},
					Type: instructionIdent(ins),
				},
			},
		},
		Name: dst.NewIdent("String"),
		Type: &dst.FuncType{
			Params: &dst.FieldList{},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Type: dst.NewIdent("string"),
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}
}

func instructionEncodeFuncDecl(ins instruction) *dst.FuncDecl {
	stmts := []dst.Stmt{
		&dst.ExprStmt{
			X: &dst.CallExpr{
				Fun: &dst.Ident{
					Name: "emitOpcode",
					Path: opcodePackagePath,
				},
				Args: []dst.Expr{
					dst.NewIdent("code"),
					&dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X:   dst.NewIdent("i"),
							Sel: dst.NewIdent("Opcode"),
						},
					},
				},
			},
			Decs: dst.ExprStmtDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.NewLine,
					After:  dst.NewLine,
				},
			},
		},
	}

	for _, operand := range ins.Operands {

		var funcName string

		switch operand.Type {
		case operandTypeBool:
			funcName = "emitBool"

		case operandTypeIndex:
			funcName = "emitUint16"

		case operandTypeIndices:
			funcName = "emitUint16Array"

		case operandTypeSize:
			funcName = "emitUint16"

		case operandTypeString:
			funcName = "emitString"

		case operandTypeCastKind:
			funcName = "emitCastKind"

		case operandTypePathDomain:
			funcName = "emitPathDomain"

		case operandTypeCompositeKind:
			funcName = "emitCompositeKind"

		default:
			panic(fmt.Sprintf("unsupported operand type: %s", operand.Type))
		}

		stmts = append(stmts,
			&dst.ExprStmt{
				X: &dst.CallExpr{
					Fun: &dst.Ident{
						Name: funcName,
						Path: opcodePackagePath,
					},
					Args: []dst.Expr{
						dst.NewIdent("code"),
						&dst.SelectorExpr{
							X:   dst.NewIdent("i"),
							Sel: operandIdent(operand),
						},
					},
				},
				Decs: dst.ExprStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Before: dst.NewLine,
						After:  dst.NewLine,
					},
				},
			},
		)
	}

	return &dst.FuncDecl{
		Recv: &dst.FieldList{
			List: []*dst.Field{
				{
					Names: []*dst.Ident{
						dst.NewIdent("i"),
					},
					Type: instructionIdent(ins),
				},
			},
		},
		Name: dst.NewIdent("Encode"),
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{
							dst.NewIdent("code"),
						},
						Type: &dst.StarExpr{
							X: &dst.ArrayType{
								Elt: dst.NewIdent("byte"),
							},
						},
					},
				},
			},
			Results: &dst.FieldList{},
		},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}
}

func instructionDecodeFuncDecl(ins instruction) *dst.FuncDecl {
	var stmts []dst.Stmt

	for _, operand := range ins.Operands {

		var funcName string

		switch operand.Type {
		case operandTypeBool:
			funcName = "decodeBool"

		case operandTypeIndex:
			funcName = "decodeUint16"

		case operandTypeIndices:
			funcName = "decodeUint16Array"

		case operandTypeSize:
			funcName = "decodeUint16"

		case operandTypeString:
			funcName = "decodeString"

		case operandTypeCastKind:
			funcName = "decodeCastKind"

		case operandTypePathDomain:
			funcName = "decodePathDomain"

		case operandTypeCompositeKind:
			funcName = "decodeCompositeKind"

		default:
			panic(fmt.Sprintf("unsupported operand type: %s", operand.Type))
		}

		stmts = append(stmts,
			&dst.AssignStmt{
				Lhs: []dst.Expr{
					&dst.SelectorExpr{
						X:   dst.NewIdent("i"),
						Sel: operandIdent(operand),
					},
				},
				Tok: token.ASSIGN,
				Rhs: []dst.Expr{
					&dst.CallExpr{
						Fun: &dst.Ident{
							Name: funcName,
							Path: opcodePackagePath,
						},
						Args: []dst.Expr{
							dst.NewIdent("ip"),
							dst.NewIdent("code"),
						},
					},
				},
				Decs: dst.AssignStmtDecorations{
					NodeDecs: dst.NodeDecs{
						Before: dst.NewLine,
						After:  dst.NewLine,
					},
				},
			},
		)
	}

	stmts = append(
		stmts,
		&dst.ReturnStmt{
			Results: []dst.Expr{
				dst.NewIdent("i"),
			},
			Decs: dst.ReturnStmtDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.NewLine,
					After:  dst.NewLine,
				},
			},
		},
	)

	return &dst.FuncDecl{
		Name: dst.NewIdent("Decode" + firstUpper(ins.Name)),
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{
							dst.NewIdent("ip"),
						},
						Type: &dst.StarExpr{
							X: dst.NewIdent("uint16"),
						},
					},
					{
						Names: []*dst.Ident{
							dst.NewIdent("code"),
						},
						Type: &dst.ArrayType{
							Elt: dst.NewIdent("byte"),
						},
					},
				},
			},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{
							dst.NewIdent("i"),
						},
						Type: instructionIdent(ins),
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}
}

func writeGoFile(writer io.Writer, decls []dst.Decl, packagePath string) {
	resolver := guess.New()
	restorer := decorator.NewRestorerWithImports(packagePath, resolver)

	packageName, err := resolver.ResolvePackage(packagePath)
	if err != nil {
		panic(err)
	}

	for _, decl := range decls {
		decl.Decorations().Before = dst.NewLine
		decl.Decorations().After = dst.EmptyLine
	}

	err = restorer.Fprint(
		writer,
		&dst.File{
			Name:  dst.NewIdent(packageName),
			Decls: decls,
		},
	)
	if err != nil {
		panic(err)
	}
}

func firstUpper(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}
