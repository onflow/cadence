import React from "react"
import {DictionaryValue, Value} from "./value.ts"
import Type from "./type.tsx"

interface Props {
    value: DictionaryValue,
    onChange?: (value: Value) => void
    onKeyDown?: (event: React.KeyboardEvent<HTMLSelectElement>) => void
}

export default function DictionaryValue({
    value,
    onChange,
    onKeyDown
}: Props) {

    function _onChange(event: React.ChangeEvent<HTMLSelectElement>) {
        const selectValue = event.target.value
        if (!selectValue)
            return;
        const index = Number(selectValue)
        const key = value.keys[index].value
        if (key.kind === "primitive") {
            onChange?.(key.value)
        }
    }

    const options = value.keys.map((key, index) =>
        <option key={index} value={index}>{key.description}</option>
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
