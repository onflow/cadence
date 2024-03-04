import React from "react"
import { CompositeValue } from "./value.ts"
import Type from "./type.tsx"

interface Props {
    value: CompositeValue
    onChange?: (field: string) => void,
    onKeyDown?: (event: React.KeyboardEvent<HTMLSelectElement>) => void
}

export default function CompositeValue({
    value,
    onChange,
    onKeyDown
}: Props) {

    function _onChange(event: React.ChangeEvent<HTMLSelectElement>) {
        const field = event.target.value
        onChange?.(field)
    }

    const options = value.fields.map(field =>
        <option key={field} value={field}>{field}</option>
    )

    return (
        <>
            <Type
                kind={value.kind}
                type={value.type}
                description={value.typeString}
            />
            <select
                size={2}
                onChange={_onChange}
                onFocus={_onChange}
                onKeyDown={onKeyDown}
            >
                {options}
            </select>
        </>
    )
}
