/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


import * as monaco from "monaco-editor";
import configureCadence, {CADENCE_LANGUAGE_ID} from "./cadence";
import {CadenceParser} from "@onflow/cadence-parser"

import * as React from "react"
import * as ReactDOM from "react-dom"
import {TreeView} from "./tree"

const code = `
pub contract C {
    pub resource R {}
    pub fun createR(): @R {
        return <- create R()
    }
}
`

interface Position {
  Offset: number
  Line: number
  Column: number
}

interface Node {
  StartPos: Position
  EndPos: Position
}

function isNode(something: unknown): something is Node {
  const node = something as Node
  return node.StartPos !== undefined && node.EndPos !== undefined
}

document.addEventListener('DOMContentLoaded', async () => {

  configureCadence()

  const editorElement = document.getElementById(`editor`);
  const astElement = document.getElementById(`ast`);

  const model = monaco.editor.createModel(
    code,
    CADENCE_LANGUAGE_ID,
    monaco.Uri.parse(`inmemory://code.cdc`)
  )

  monaco.editor.create(
    editorElement,
    {
      theme: 'vs-light',
      language: CADENCE_LANGUAGE_ID,
      model: model,
      minimap: {
        enabled: false
      },
    }
  );

  const parser = await CadenceParser.create("cadence-parser.wasm")

  function update() {
    const code = model.getValue()
    const result = parser.parse(code)
    console.log(result)

    let decorations: string[];

    let current: unknown;

    ReactDOM.render(
      <TreeView
        data={result}
        onOver={node => {
          if (!isNode(node)) {
            return false
          }
          current = node
          decorations = model.deltaDecorations(decorations, [
            {
              range: new monaco.Range(
                node.StartPos.Line,
                node.StartPos.Column + 1,
                node.EndPos.Line,
                node.EndPos.Column + 2
              ),
              options: {
                inlineClassName: 'highlighted'
              }
            },
          ]);
          return true
        }}
        onLeave={node => {
          if (node === current) {
            decorations = model.deltaDecorations(decorations, [])
          }
        }}
      />,
      astElement
    )
  }

  model.onDidChangeContent(update)

  update()
})
