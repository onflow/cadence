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
          "Access": "AccessPublic",
          "DocString": "",
          "EndPos": {
            "Column": 16,
            "Line": 1,
            "Offset": 16,
          },
          "FunctionBlock": {
            "Block": {
              "EndPos": {
                "Column": 16,
                "Line": 1,
                "Offset": 16,
              },
              "StartPos": {
                "Column": 15,
                "Line": 1,
                "Offset": 15,
              },
              "Statements": null,
              "Type": "Block",
            },
            "EndPos": {
              "Column": 16,
              "Line": 1,
              "Offset": 16,
            },
            "StartPos": {
              "Column": 15,
              "Line": 1,
              "Offset": 15,
            },
            "Type": "FunctionBlock",
          },
          "Identifier": {
            "EndPos": {
              "Column": 11,
              "Line": 1,
              "Offset": 11,
            },
            "Identifier": "main",
            "StartPos": {
              "Column": 8,
              "Line": 1,
              "Offset": 8,
            },
          },
          "ParameterList": {
            "EndPos": {
              "Column": 13,
              "Line": 1,
              "Offset": 13,
            },
            "Parameters": null,
            "StartPos": {
              "Column": 12,
              "Line": 1,
              "Offset": 12,
            },
          },
          "ReturnTypeAnnotation": {
            "AnnotatedType": {
              "EndPos": {
                "Column": 12,
                "Line": 1,
                "Offset": 12,
              },
              "Identifier": {
                "EndPos": {
                  "Column": 12,
                  "Line": 1,
                  "Offset": 12,
                },
                "Identifier": "",
                "StartPos": {
                  "Column": 13,
                  "Line": 1,
                  "Offset": 13,
                },
              },
              "StartPos": {
                "Column": 13,
                "Line": 1,
                "Offset": 13,
              },
              "Type": "NominalType",
            },
            "EndPos": {
              "Column": 12,
              "Line": 1,
              "Offset": 12,
            },
            "IsResource": false,
            "StartPos": {
              "Column": 13,
              "Line": 1,
              "Offset": 13,
            },
          },
          "StartPos": {
            "Column": 0,
            "Line": 1,
            "Offset": 0,
          },
          "Type": "FunctionDeclaration",
        },
      ],
    },
  })
})
