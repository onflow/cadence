import React from "react"
import { CompositeValue } from "./value.ts"

interface Props {
    keyPath: string[]
    value: CompositeValue
}

export default function CompositeValue({
    keyPath,
    value
}: Props) {
    const key = keyPath[keyPath.length - 1]

    return (
        <>
            <h2>{key}</h2>
            <select size={2}>
                {value.fields.map(field => (
                    <option key={field} value={field}>{field}</option>
                ))}
            </select>
        </>
    )
}
