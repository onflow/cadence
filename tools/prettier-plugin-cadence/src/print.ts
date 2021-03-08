import { doc, Doc, FastPath } from "prettier"
import {
	Access,
	Location,
	Node,
	Parameter,
	TypeAnnotation,
	ImportDeclaration,
} from "./nodes"
import { isAddressLocation } from "./typeCheck"
import concat = doc.builders.concat
import hardline = doc.builders.hardline
import join = doc.builders.join
import group = doc.builders.group;

const SPACE = " " // single space character
const COMMA = ", "

function getIdentifier(item: any): string {
	return item.Identifier.Identifier
}

function accessToString(access: Access) {
	switch (access) {
		case "AccessPublic":
			return "pub"
		case "AccessAccount":
			return "access(account)"
		case "AccessContract":
			return "access(contract)"
		case "AccessNotSpecified":
			return ""
		default:
			return ""
	}
}

function locationToString(location: Location) {
	return isAddressLocation(location)
		? location.Address
		: `"${location.String}"`
}

function importToString(importNode: ImportDeclaration) {
	const contractList = join(
		COMMA,
		importNode.Identifiers.map((id) => id.Identifier)
	)
	const fromTarget = locationToString(importNode.Location)
	return join(SPACE, ["import", contractList, "from", fromTarget])
}

function typeToString(typeAnnotation: TypeAnnotation) {
	const id = getIdentifier(typeAnnotation.AnnotatedType)
	const prefix = id === "" ? "" : ": "
	const isResource = typeAnnotation.IsResource ? "@" : ""
	return concat([prefix, isResource, id])
}

function getParameterName(parameter: Parameter) {
	const suffix = parameter.Label ? `${parameter.Label}` : ""
	return concat([suffix, getIdentifier(parameter)])
}

function makeParametersList(parameterList: Parameter[]) {
	const list = parameterList
		? join(
				", ",
				parameterList.map((parameter: Parameter) => {
					return concat([
						getParameterName(parameter),
						typeToString(parameter.TypeAnnotation),
					])
				})
		  )
		: ""
	return concat(["(", list, ")"])
}

export function print(
	path: FastPath<Node>,
	options: object,
	print: (path: FastPath) => Doc
): Doc | null {
	const n = path.getValue()

	switch (n.Type) {
		case "StringExpression":
			return concat(['"', n.Value, '"'])

		case "NilExpression":
			return "nil"

		case "Program":
			const parts = path.map(print, "Declarations")

			// Only force a trailing newline if there were any contents.
			if (n.Declarations.length) {
				parts.push(hardline)
			}

			return join(hardline, parts)

		case "ImportDeclaration":
			return importToString(n)

		case "ImportGroup":
			const result = join(hardline, n.Declarations.map(importToString));
			return concat([result, hardline])

		case "FunctionDeclaration":
			const access = concat([accessToString(n.Access), SPACE])
			let returnType = concat([typeToString(n.ReturnTypeAnnotation), " "])
			const parameterList = makeParametersList(n.ParameterList.Parameters)
			return concat([
				access,
				"fun ",
				n.Identifier.Identifier,
				parameterList,
				returnType,
				"{",
				"}",
			])

		default:
			const some: { Type: string } = n
			throw new Error(`unsupported AST node: ${some.Type}`)
	}
}
