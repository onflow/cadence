import React from "react"
import styles from "./type.module.css"
import { Type } from "./value.ts"

type Kind = "dictionary" | "composite" | "array" | "primitive" | "fallback"

type Category = "container" | "leaf"

interface Props {
    kind: Kind
    type: Type
    description: string
}

function category(kind: Kind): Category {
    switch (kind) {
        case "dictionary":
        case "composite":
        case "array":
            return "container"
        case "primitive":
        case "fallback":
            return "leaf"
    }
}

export default function Type({
    kind,
    description
}: Props) {
    return (
        <div className={`${styles.type} ${styles[`type-${category(kind)}`]}`}>
            {description}
        </div>
    )
}
