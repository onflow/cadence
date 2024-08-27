import {CadenceParser} from "../src"
import * as fs from "fs"

test("parse simple", async () => {
  const binary = fs.readFileSync(require.resolve('../dist/cadence-parser.wasm'))
  const parser = await CadenceParser.create(binary)
  const res = parser.parse("access(all) fun main() {}")
  expect(res).toEqual({
    "program": {
      "Type": "Program",
      "Declarations": [
        {
          "Access": "AccessAll",
          "DocString": "",
          "EndPos": {
            "Column": 24,
            "Line": 1,
            "Offset": 24,
          },
          "FunctionBlock": {
            "Block": {
              "EndPos": {
                "Column": 24,
                "Line": 1,
                "Offset": 24,
              },
              "StartPos": {
                "Column": 23,
                "Line": 1,
                "Offset": 23,
              },
              "Statements": null,
              "Type": "Block",
            },
            "EndPos": {
              "Column": 24,
              "Line": 1,
              "Offset": 24,
            },
            "StartPos": {
              "Column": 23,
              "Line": 1,
              "Offset": 23,
            },
            "Type": "FunctionBlock",
          },
          "Identifier": {
            "EndPos": {
              "Column": 19,
              "Line": 1,
              "Offset": 19,
            },
            "Identifier": "main",
            "StartPos": {
              "Column": 16,
              "Line": 1,
              "Offset": 16,
            },
          },
          "IsNative": false,
          "IsStatic": false,
          "ParameterList": {
            "EndPos": {
              "Column": 21,
              "Line": 1,
              "Offset": 21,
            },
            "Parameters": null,
            "StartPos": {
              "Column": 20,
              "Line": 1,
              "Offset": 20,
            },
          },
          "Purity": "Unspecified",
          "ReturnTypeAnnotation": null,
          "StartPos": {
            "Column": 0,
            "Line": 1,
            "Offset": 0,
          },
          "Type": "FunctionDeclaration",
          "TypeParameterList": null,
        },
      ],
    },
  })
})
