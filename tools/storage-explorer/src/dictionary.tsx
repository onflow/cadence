import React from "react"
import { DictionaryValue } from "./value.ts"

interface Props {
    keyPath: string[]
    value: DictionaryValue
}

export default function DictionaryValue({
    keyPath,
    value
}: Props) {
    const key = keyPath[keyPath.length - 1]

    return (
        <>
            <h2>{key}</h2>
            <select size={2}>
                {value.keys.map((key, i) => (
                    <option key={i} value={i}>{key+""}</option>
                ))}
            </select>
        </>
    )
}
