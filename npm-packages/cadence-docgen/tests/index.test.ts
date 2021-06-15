import {CadenceDocgen} from "../src"
import * as fs from "fs"

test("docgen simple", async () => {
  const binary = fs.readFileSync(require.resolve('../dist/cadence-docgen.wasm'))
  const docgen = await CadenceDocgen.create(binary)

  const res = docgen.generate(`
    /// This is a 'Foo' contract.
    contract Foo {
    }
  `)

  expect(res).toEqual({
    "docs": {
      "Foo.md": "# Contract `Foo`\n\n```cadence\ncontract Foo\n}\n```\n\nThis is a 'Foo' contract.\n"
    }
  })
})
