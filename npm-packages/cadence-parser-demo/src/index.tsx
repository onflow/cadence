/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
import {CadenceParser} from "cadence-parser"

import * as React from "react"
import * as ReactDOM from "react-dom"
import ReactJson from "react-json-view"

const code = `
pub contract C {

    pub resource R {}

    pub fun createR(): @R {
        return <- create R()
    }
}
`

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

    ReactDOM.render(
      <ReactJson
        src={result}
        enableClipboard={false}
        displayDataTypes={false}

      />,
      astElement
    )
  }

  model.onDidChangeContent(update)

  update()
})
