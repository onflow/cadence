import {doc, Doc, FastPath} from "prettier";
import {Node} from "./nodes";
import concat = doc.builders.concat;
import hardline = doc.builders.hardline;
import join = doc.builders.join;
import group = doc.builders.group;

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
      return concat([
        "fun ",
        n.Identifier.Identifier,
        group(concat(['(', ')'])),
        " ",
        "{",
        "}"
      ])

    default:
      const some: {Type: string} = n
      throw new Error(`unsupported AST node: ${some.Type}`)
  }
}
