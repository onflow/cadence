/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
import * as React from "react"
import { Data, TreeView } from "./tree";
import { createRoot } from 'react-dom/client';

const defaultCode = `
pub contract C {
    pub resource R {}
    pub fun createR(): @R {
        return <- create R()
    }
}
`

function Error({ children }: {children?: React.ReactNode[]}) {
  return <div className='error'>{children}</div>;
}

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

  const code = localStorage.getItem('code') || defaultCode

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

  const root = createRoot(astElement)
  const render = root.render.bind(root)

  async function update() {
    const code = model.getValue()
    localStorage.setItem('code', code)

    const request = new Request('/api', {
      method: 'POST',
      body: JSON.stringify({code})
    });

    let response: Response
    try {
      response = await fetch(request)
    } catch (e) {
      render(
        <Error>💥 Failed to make API request: {e.toString()}</Error>
      )
      return
    }

    let result: Data
    try {
      result = await response.json()
    } catch (e) {
      render(
        <Error>💥 Failed to parse result as JSON: {e.toString()}</Error>
      )
      return
    }

    if (result.error) {
      render(
        <Error>💥 {result.error.toString()}</Error>
      )
      return
    }


    let decorations: string[];

    let current: unknown;

    render(
      <TreeView
        data={result.program}
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
    )
  }

  model.onDidChangeContent(update)

  update()
})
