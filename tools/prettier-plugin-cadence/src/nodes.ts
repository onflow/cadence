
export type Node =
  | Program
  | Declaration
  | Expression

export interface Identifier {
  Identifier: string
}

export interface Program {
  Type: "Program"
  Declarations: Declaration[]
}

// Declarations

export type Declaration =
  FunctionDeclaration

export interface FunctionDeclaration {
  Type: "FunctionDeclaration"
  Identifier: Identifier
}

// Expressions

export type Expression =
  StringExpression
  | NilExpression

export interface NilExpression {
  Type: "NilExpression"
}

interface StringExpression {
  Type: "StringExpression"
  Value: string
}
