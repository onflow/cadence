# FAQ

## Is there a formal grammar (e.g. in BNF) for Cadence?

Yes, there is a [EBNF for Cadence](https://github.com/onflow/cadence/blob/master/docs/cadence.ebnf).

## How can we get syntax highlighting on GitHub?

Syntax highlighting for GitHub is implemented in the [`linguist`](https://github.com/github/linguist) library.
Linguist supports TextMate grammars, and we already have a [TextMate grammar for Cadence in the Visual Studio Code extension](https://github.com/onflow/vscode-flow/blob/master/syntaxes/cadence.tmGrammar.json).

However, GitHub ["[tries] to add new extensions once they have some usage on GitHub. In most cases [GitHub] prefer[s] that each new file extension be in use in at least 200 unique :user/:repo repositories before supporting them [...]"](https://github.com/github/linguist/blob/master/CONTRIBUTING.md#adding-an-extension-to-a-language).

Once we have reached this threshold for Cadence, we should add support for Cadence to linguist.

## How can I inject additional values when executing a transaction or script?

The runtime `Interface` functions `ExecuteTransaction` and `ExecuteScript` require a `Context` argument.
The context has a `PredeclaredValues` field, which can be filled with `ValueDeclaration` values.

Optionally, value declarations may have a predicate function `Available` of type `func(common.Location) bool`.
The checker calls this function for every location that is checked,
to determine if the value declaration should be available/declared in the given location.

For example, this allows declaring a function that is only available in the service account.
In this case, the availability function would need to check if the location is an `AddressLocation`,
and that the address of the address location is the address of the service account.

## How is Cadence parsed?

Cadence's parser is implemented as a hand-written recursive descent parser which uses operator precedence parsing.
The recursive decent parsing technique allows for greater control, e.g. when implementing whitespace sensitivity, ambiguities, etc.
The handwritten parser also allows for better / great custom error reporting and recovery.

The operator precedence parsing technique avoids constructing a CST and the associated overhead, where each grammar rule is translated to a CST node.
For example, a simple integer literal would be "boxed" in several outer grammar rule nodes.
