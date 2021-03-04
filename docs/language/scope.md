---
title: Scope
type: REF
---

Every function and block (`{` ... `}`) introduces a new scope for declarations.
Each function and block can refer to declarations in its scope or any of the outer scopes.

```cadence
let x = 10

fun f(): Int {
    let y = 10
    return x + y
}

f()  // is `20`

// Invalid: the identifier `y` is not in scope.
//
y
```

```cadence
fun doubleAndAddOne(_ n: Int): Int {
    fun double(_ x: Int) {
        return x * 2
    }
    return double(n) + 1
}

// Invalid: the identifier `double` is not in scope.
//
double(1)
```

Each scope can introduce new declarations, i.e., the outer declaration is shadowed.

```cadence
let x = 2

fun test(): Int {
    let x = 3
    return x
}

test()  // is `3`
```

Scope is lexical, not dynamic.

```cadence
let x = 10

fun f(): Int {
   return x
}

fun g(): Int {
   let x = 20
   return f()
}

g()  // is `10`, not `20`
```

Declarations are **not** moved to the top of the enclosing function (hoisted).

```cadence
let x = 2

fun f(): Int {
    if x == 0 {
        let x = 3
        return x
    }
    return x
}
f()  // is `2`
```

