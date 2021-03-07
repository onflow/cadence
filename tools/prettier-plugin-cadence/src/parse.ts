import {AST, Parser, ParserOptions} from "prettier";
import * as execa from "execa"

export function parse(
  text: string,
  parsers: { [parserName: string]: Parser },
  opts: ParserOptions,
): AST | null {
  const returnValue = execa.sync("./parse", ['--json'], {input: text});
  const result = JSON.parse(returnValue.stdout)[0]
  if (result.Error) {
    return null
  }
  return result.program
}
