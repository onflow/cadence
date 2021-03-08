import { AST } from "prettier"

export function preprocess(ast: AST, options: object): AST {
	const declarations = ast.Declarations.reduce(
		(acc: any, declaration: any, i: number, arr: any) => {
			if (declaration.Type === "ImportDeclaration") {
				acc.importGroup["Declarations"].push(declaration)
				const lastItem = i === arr.length - 1
				if (lastItem){
					acc.result.push(acc.importGroup)
				}
			} else {
				if (
					!acc.importsProcessed &&
					acc.importGroup.Declarations.length > 0
				) {
					acc.importsProcessed = true
					acc.result.push(acc.importGroup)
				}
				acc.result.push(declaration)
			}
			return acc
		},
		{
			importGroup: {
				Type: "ImportGroup",
				Declarations: [],
			},
			importsProcessed: false,
			result: [],
		}
	)
	ast.Declarations = declarations.result
	return ast
}
