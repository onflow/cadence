import exp from "constants"

export type Node = Program | Declaration | Expression

export type Access =
	| "AccessNotSpecified"
	| "AccessPrivate"
	| "AccessContract"
	| "AccessAccount"
	| "AccessPublic"
	| "AccessPublicSettable"

export type Type =
	| "NominalType"
	| "OptionalType"
	| "VariableSizedType"
	| "ConstantSizedType"
	| "DictionaryType"
	| "FunctionType"
	| "ReferenceType"
	| "RestrictedType"
	| "InstantiationType"

export interface Identifier {
	Identifier: string
}

export interface Program {
	Type: "Program"
	Declarations: Declaration[]
}

export interface AnnotatedType {
	Identifier: Identifier
	Type: Type
}

export interface TypeAnnotation {
	IsResource: boolean
	AnnotatedType: AnnotatedType
}

// Declarations

export type Declaration = FunctionDeclaration

export interface FunctionDeclaration {
	Type: "FunctionDeclaration"
	Identifier: Identifier
	Access: Access
	ParameterList: ParameterList
	ReturnTypeAnnotation: TypeAnnotation
}

export interface Parameter {
	Label: string
	Identifier: Identifier
	TypeAnnotation: TypeAnnotation
}

export interface ParameterList {
	Parameters: Parameter[]
}

// Expressions

export type Expression = StringExpression | NilExpression

export interface NilExpression {
	Type: "NilExpression"
}

interface StringExpression {
	Type: "StringExpression"
	Value: string
}
