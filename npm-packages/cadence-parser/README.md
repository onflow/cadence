# Cadence Parser

The [Cadence](https://github.com/onflow/cadence) parser compiled to WebAssembly and bundled as an NPM package,
so it can be used in tools written in JavaScript.

## Usage

```js
import {CadenceParser} from "@onflow/cadence-parser"

const parser = await CadenceParser.create("cadence-parser.wasm")

const ast = parser.parse(`
  pub contract HelloWorld {
    pub fun hello() {
      log("Hello, world!")
    }
  }
`)
```
