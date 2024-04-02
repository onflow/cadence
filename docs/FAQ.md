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

## What is the algorithmic efficiency of operations on arrays and dictionaries?

Arrays and dictionaries are implemented [as trees](https://github.com/onflow/atree).
This means that lookup operations do not run in constant time.
In certain cases, a mutation operation may cause a rebalancing of the tree.

## Analyzing Cadence code

To analyze Cadence code, you can use the [Go package `github.com/onflow/cadence/analysis`](https://github.com/onflow/cadence/tree/master/tools/analysis).
It is similar to the [Go package `golang.org/x/tools/go/analysis`](https://pkg.go.dev/golang.org/x/tools/go/analysis), which allows analyzing Go code.
The blog post at https://eli.thegreenplace.net/2020/writing-multi-package-analysis-tools-for-go/ can be followed to learn more about how to write an analyzer.
The API of the analysis package for Cadence programs is fairly similar to the package for Go programs, but not identical.

To run the analyzer pass, the [Cadence linter tool](https://github.com/onflow/cadence-tools/tree/master/lint#cadence-lint) can be used.
For example, it allows running the pass over all contracts of a network.

There are several options to run the analyzer pass with the linter tool:
- The analysis pass can be written directly in the linter tool. See the existing passes in the linter tool for examples.
- The analysis pass can be written in a separate package in Cadence, and the linter tool can use it.
  The go.mod `replace` statement can be used to point to a locally modified Cadence working directory.
- The linter supports [Go plugins](https://pkg.go.dev/plugin) (see e.g. https://eli.thegreenplace.net/2021/plugins-in-go/) https://github.com/onflow/cadence-tools/blob/83eb7d4d19ddf2dd7ad3fdcc6aa6451a6bc126ff/lint/cmd/lint/main.go#L48.
  The analyzer pass can be written in a separate module, built as a plugin, and loaded in the linter using the `-plugin` command line option.

## Analyzing Cadence values / state snapshots

To analyze Cadence values (`interpreter.Value`), you can use the function [`interpreter.InspectValue`](https://github.com/onflow/cadence/blob/master/runtime/interpreter/inspect.go#L31).

To find static types in Cadence values (e.g. in type values, containers, capabilities), you can see which values contain static types in the [Cadence 1.0 static type migration code](https://github.com/onflow/cadence/blob/master/migrations/statictypes/statictype_migration.go#L67).

To load values from a state snapshot you can use the [flow-go `util` commad](https://github.com/onflow/flow-go/tree/master/cmd/util) to convert a state snapshot in trie format to a file which just contains the payloads.

To get a `runtime.Storage` instance from it, use `util.ReadPayloadFile`, `util.NewPayloadSnapshot`, `state.NewTransactionState`, `environment.NewAccounts`, `util.NewAccountsAtreeLedger`, and finally `runtime.NewStorage`.
