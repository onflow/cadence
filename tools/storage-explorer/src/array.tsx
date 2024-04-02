import React, {ReactNode} from "react"
import { ArrayValue } from "./value.ts"
import Type from "./type.tsx"

interface Props {
    value: ArrayValue,
    onChange?: (index: number) => void,
    onKeyDown?: (event: React.KeyboardEvent<HTMLSelectElement>) => void
}

export default function ArrayValue({
    value,
    onChange,
    onKeyDown
}: Props) {

    function _onChange(event: React.ChangeEvent<HTMLSelectElement>) {
        const index = Number(event.target.value)
        if (index < value.count)
            onChange?.(index)
    }

    const options: ReactNode[] = Array.from({ length: value.count })
    for (let i = 0; i < value.count; i++) {
        options.push(<option key={i} value={i}>{i}</option>)
    }

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
