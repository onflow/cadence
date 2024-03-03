import React from "react"
import { CompositeValue } from "./value.ts"
import Type from "./type.tsx"

interface Props {
    value: CompositeValue
    onChange?: (field: string) => void
}

export default function CompositeValue({
    value,
    onChange
}: Props) {

    function _onChange(event: React.ChangeEvent<HTMLSelectElement>) {
        const field = event.target.value
        onChange?.(field)
    }

    return (
        <>
            <Type
                kind={value.kind}
                type={value.type}
                description={value.typeString}
            />
            <select size={2} onChange={_onChange}>
                {value.fields.map(field => (
                    <option key={field} value={field}>{field}</option>
                ))}
            </select>
        </>
    )
}
