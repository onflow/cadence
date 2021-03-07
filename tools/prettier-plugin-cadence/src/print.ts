import {doc, Doc, FastPath} from "prettier";
import {Access, AnnotatedType, Identifier, Node, Parameter, ParameterList, TypeAnnotation} from "./nodes";
import concat = doc.builders.concat;
import hardline = doc.builders.hardline;
import join = doc.builders.join;
import group = doc.builders.group;

// overload this for different types
function getIdentifier(annotatedType: AnnotatedType): string
function getIdentifier(parameter: Parameter): string
function getIdentifier(item: any): string {
  return item.Identifier.Identifier
}

function accessToString(access: Access){
  let result = ""
  switch (access){
    case "AccessPublic":
      result = "pub"
      break;
    case "AccessAccount":
      result = "access(account)"
      break;
    case "AccessContract":
      result = "access(contract)"
      break;
    case "AccessNotSpecified":
      result = ""
      break
    default:
      result = ""
  }

  return `${result} `
}

function typeToString(typeAnnotation: TypeAnnotation){
  const id = getIdentifier(typeAnnotation.AnnotatedType);
  const prefix = id === "" ? "" : ": "
  const isResource = typeAnnotation.IsResource ? "@" : ""
  return concat([prefix, isResource, id])
}

function getParameterName(parameter: Parameter){
  const suffix = parameter.Label ? `${parameter.Label }` : ''
  return concat([suffix, getIdentifier(parameter)])
}

function makeParametersList(parameterList: Parameter[]){
  const list = parameterList
      ? join(', ', parameterList.map((parameter: Parameter)=> {
        return concat([
          getParameterName(parameter),
          typeToString(parameter.TypeAnnotation)])
      }))
      : ""
  return concat(['(', list, ')'])
}

export function print(
  path: FastPath<Node>,
  options: object,
  print: (path: FastPath) => Doc
): Doc | null {
  const n = path.getValue();

  switch (n.Type) {
    case "StringExpression":
      return concat(['"', n.Value, '"'])

    case "NilExpression":
      return "nil"

    case "Program":
      const parts = path.map(print, 'Declarations')

      // Only force a trailing newline if there were any contents.
      if (n.Declarations.length) {
        parts.push(hardline);
      }

      return join(hardline, parts)

    case "FunctionDeclaration":
      const access = accessToString(n.Access)
      let returnType = concat([typeToString(n.ReturnTypeAnnotation)," "])
      const parameterList = makeParametersList(n.ParameterList.Parameters)
      return concat([
        access,
        "fun ",
        n.Identifier.Identifier,
        parameterList,
        returnType,
        "{",
        "}"
      ])

    default:
      const some: {Type: string} = n
      throw new Error(`unsupported AST node: ${some.Type}`)
  }
}
