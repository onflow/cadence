# FAQ

## Is there a formal grammar (e.g. in BNF) for Cadence?

Yes, there is a [EBNF for Cadence](https://github.com/onflow/cadence/blob/master/docs/cadence.ebnf).

## How can we get syntax highlighting on GitHub?

Syntax highlighting for GitHub is implemented in the [`linguist`](https://github.com/github/linguist) library.
Linguist supports TextMate grammars, and we already have a [TextMate grammar for Cadence in the Visual Studio Code extension](https://github.com/onflow/vscode-flow/blob/master/syntaxes/cadence.tmGrammar.json).

However, GitHub "[...] prefers that each new file extension be in use in hundreds of repositories before supporting them in Linguist".
Once we have reached this threshold for Cadence, we should add support for Cadence to linguist.
