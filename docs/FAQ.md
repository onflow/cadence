# FAQ

## Is there a formal grammar (e.g. in BNF) for Cadence?

Yes, there is a [EBNF for Cadence](https://github.com/onflow/cadence/blob/master/docs/cadence.ebnf).

## How can I inject additional values when executing a transaction or script?

The runtime `Interface` functions `ExecuteTransaction` and `ExecuteScript` require a `Context` argument.
The context has an `Environment` field, in which `stdlib.StandardLibraryValue`s can be declared.

## How is Cadence parsed?

Cadence's parser is implemented as a hand-written recursive descent parser which uses operator precedence parsing.
The recursive decent parsing technique allows for greater control, e.g. when implementing whitespace sensitivity, ambiguities, etc.
The handwritten parser also allows for better / great custom error reporting and recovery.

The operator precedence parsing technique avoids constructing a CST and the associated overhead, where each grammar rule is translated to a CST node.
For example, a simple integer literal would be "boxed" in several outer grammar rule nodes.

## What is the algorithm efficiency of operations on arrays and dictionaries?

Arrays and dictionaries are implemented [as trees](https://github.com/onflow/atree). 
This means that lookup operations are not constant time. 
In certain cases, a mutation operation may cause a rebalancing of the tree.
