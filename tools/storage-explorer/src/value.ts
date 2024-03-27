import CompositeValue from "./composite.tsx"

export type Type = string | any

export interface CompositeValue {
    kind: "composite"
    type: Type
    typeString: string
    fields: string[]
}

export interface DictionaryValue {
    kind: "dictionary"
    type: Type
    typeString: string
    keys: DictionaryKey[]
}

interface DictionaryKey {
    description: string
    value: Value
}

export interface ArrayValue {
    kind: "array"
    type: Type
    typeString: string
    count: number
}

export interface PrimitiveValue {
    kind: "primitive"
    type: Type
    typeString: string
    value: any
    description: string
}

export interface FallbackValue {
    kind: "fallback"
    type: Type
    typeString: string
    description: string
}

export type Value = CompositeValue | DictionaryValue | ArrayValue | PrimitiveValue | FallbackValue
