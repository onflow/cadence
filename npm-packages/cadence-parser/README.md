# Cadence Parser

The [Cadence](https://github.com/onflow/cadence) parser compiled to WebAssembly and bundled as an NPM package,
so it can be used in tools written in JavaScript.

## Usage

Broswer

```js
import { CadenceParser } from "@onflow/cadence-parser"

const parser = await CadenceParser.create("cadence-parser.wasm")

const ast = parser.parse(`
  pub contract HelloWorld {
    pub fun hello() {
      log("Hello, world!")
    }
  }
`)
```

Node.js

```js
const { CadenceParser } = require("@onflow/cadence-parser");
const fs = require("fs");
const path = require("path");

(async () => {
  const parser = await CadenceParser.create(
    await fs.promises.readFile(
      path.join(
        __dirname,
        "./node_modules/@onflow/cadence-parser/dist/cadence-parser.wasm"
      )
    )
  );

  const ast = parser.parse(`
  pub contract HelloWorld {
    pub fun hello() {
      log("Hello, world!")
    }
  }
`);
})();
```
