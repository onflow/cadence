import CompositeValue from "./composite.tsx";

export interface CompositeValue {
    kind: "composite"
    fields: string[]
}

export interface DictionaryValue {
    kind: "dictionary"
    keys: Value[]
}

export type Value = CompositeValue | DictionaryValue
