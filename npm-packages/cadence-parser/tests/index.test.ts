import {CadenceParser} from "../src"
import * as fs from "fs"

test("parse simple", async () => {
  const binary = fs.readFileSync(require.resolve('../dist/cadence-parser.wasm'))
  const parser = await CadenceParser.create(binary)
  const res = parser.parse("pub fun main() {}")
  expect(res).toEqual({
    "program": {
      "Type": "Program",
        "Declarations": [
          {
          "TypeParameterList": null,
          "ParameterList": {
            "Parameters": null,
            "StartPos": {
              "Offset": 12,
              "Line": 1,
              "Column": 12
            },
            "EndPos": {
              "Offset": 13,
              "Line": 1,
              "Column": 13
            }
          },
          "ReturnTypeAnnotation": null,
          "FunctionBlock": {
            "Block": {
              "Statements": null,
              "StartPos": {
                "Offset": 15,
                "Line": 1,
                "Column": 15
              },
              "EndPos": {
                "Offset": 16,
                "Line": 1,
                "Column": 16
              },
              "Type": "Block"
            },
            "Type": "FunctionBlock",
            "StartPos": {
              "Offset": 15,
              "Line": 1,
              "Column": 15
            },
            "EndPos": {
              "Offset": 16,
              "Line": 1,
              "Column": 16
            }
          },
          "DocString": "",
          "Identifier": {
            "Identifier": "main",
            "StartPos": {
              "Offset": 8,
              "Line": 1,
              "Column": 8
            },
            "EndPos": {
              "Offset": 11,
              "Line": 1,
              "Column": 11
            }
          },
          "Access": "AccessPublic",
          "Type": "FunctionDeclaration",
          "StartPos": {
            "Offset": 0,
            "Line": 1,
            "Column": 0
          },
          "EndPos": {
            "Offset": 16,
            "Line": 1,
            "Column": 16
          },
          "IsStatic": false,
          "IsNative": false
        }
      ],
    },
  })
})
