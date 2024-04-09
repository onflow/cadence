import React from "react"
import { FallbackValue } from "./value.ts"
import Type from "./type.tsx"

interface Props {
    value: FallbackValue,
}

export default function FallbackValue({
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
