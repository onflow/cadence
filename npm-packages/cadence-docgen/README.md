# Cadence Documentation Generator

The [Cadence](https://github.com/onflow/cadence) docgen tool compiled to WebAssembly and bundled as an NPM package,
so it can be used in tools written in JavaScript.

## Usage

```js
import {CadenceDocgen} from "@onflow/cadence-docgen"

const docgen = await CadenceDocgen.create("cadence-docgen.wasm")

const docs = docgen.generate(`
  /// This is a simple function with a doc-comment.
  pub fun hello() {
  }
`)
```

