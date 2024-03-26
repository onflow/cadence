import React from "react"
import { PrimitiveValue } from "./value.ts"
import Type from "./type.tsx"

interface Props {
    value: PrimitiveValue,
}

export default function PrimitiveValue({
    value,
}: Props) {
    return (
        <>
            <Type
                kind={value.kind}
                type={value.type}
                description={value.typeString}
            />
            <pre>{value.description}</pre>
        </>
    )
}
