# Syntax Highlighting Cadence

There are currently several options to highlight Cadence code.
Currently those options are integrated into the projects they are used in, but they could be extracted and made generally useable (please let us know e.g. by creating a feature request issue).

## HTML output

If highlighted Cadence code is needed as HTML output, then a highlighter based on a [TextMate grammar for Cadence](https://github.com/onflow/flow/blob/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-theme-flow/cadence.tmGrammar.json) can be used.

This option is used by the Flow documentation: Code fences with Cadence code in the Markdown documents are converted to HTML using a [plugin](https://github.com/onflow/flow/tree/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-remark-vscode-flow).
Part of the plugin is a [highlighter class](https://github.com/onflow/flow/blob/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-remark-vscode-flow/highlighter.js) which was written to be fairly self-standing, takes Cadence code as input, and returns [hast](https://github.com/syntax-tree/hast), which is then [further converted to HTML using the `hast-util-to-html` package](https://github.com/onflow/flow/blob/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-remark-vscode-flow/index.js#L59-L77).

Another option to use this grammar is to use https://github.com/wooorm/starry-night.

## Editor

Cadence code can also be highlighted in an editor like [Monaco](https://microsoft.github.io/monaco-editor/) (which is the editor library used in Visual Studio Code), potentially in a read-only mode.

This option is currently used in the [Flow Playground](https://play.onflow.org/).

The Monaco editor does not support TextMate grammars and has its [own grammar format Monarch](https://microsoft.github.io/monaco-editor/monarch.html), so a [separate Monarch grammar for Cadence](https://github.com/onflow/flow-playground/blob/79657ebaf8682695c89c028c3bed91c780633666/src/util/cadence.ts#L15-L194) exists.
