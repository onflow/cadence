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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/parser2"
)

func pretty(code string, maxLineWidth int) string {
	program, err := parser2.ParseProgram(code, nil)
	if err != nil {
		return err.Error()
	}

	declarations := program.Declarations()

	docs := make([]prettier.Doc, 0, len(declarations))

	for _, declaration := range declarations {
		// TODO: replace once Declaration implements Doc
		hasDoc, ok := declaration.(interface{ Doc() prettier.Doc })
		if !ok {
			continue
		}

		docs = append(docs, hasDoc.Doc())
	}

	var b strings.Builder
	prettier.Prettier(&b, prettier.Concat(docs), maxLineWidth, "    ")
	return b.String()
}

//language=html
const page = `
<html>
<head>
    <title>Pretty</title>
    <style>
        :root {
            --line-length: 0ch;
        }

        body {
            margin: 0;
            padding: 0;
            font-family: monospace;
            height: 100vh;
            overflow: hidden;
        }

        #panels {
            display: grid;
            height: 100%;
            grid-template-rows: 1fr;
            grid-template-columns: 50% 50%;
            grid-template-areas: "editor ast";
        }

        #editor {
            grid-area: editor;
            border: 1px solid #ccc;
            resize: none;
        }

        #pretty {
            position: relative;
            grid-area: ast;
        }

        #output {
            white-space: pre;
        }

        #bar {
            position: absolute;
            left: var(--line-length);
            top: 0;
            bottom: 0;
            width: 2px;
            background-color: black;
        }

    </style>
</head>
<body id="panels">
<textarea id="editor" onkeydown="if(event.keyCode===9){var v=this.value,s=this.selectionStart,e=this.selectionEnd;this.value=v.substring(0, s)+'    '+v.substring(e);this.selectionStart=this.selectionEnd=s+4;return false;}"></textarea>
<div id="pretty">
    <input id="stepper" type="number" min="1" step="1">
    <div id="output"></div>
    <div id="bar"></div>
</div>
</body>
<script>
    let code = localStorage.getItem('code') || ''
    let maxLineLength = Number(localStorage.getItem('maxLineLength')) || 80;

    const root = document.documentElement;
    const editor = document.getElementById("editor")
    const output = document.getElementById("output")
    const stepper = document.getElementById("stepper")

    document.addEventListener('DOMContentLoaded', () => {
        stepper.value = maxLineLength
        editor.innerHTML = code
        update()
    })

    editor.addEventListener("input", (e) => {
        code = e.target.value
        localStorage.setItem('code', code)
        update()
    })

    stepper.addEventListener("input", (e) => {
        maxLineLength = Number(e.target.value)
        localStorage.setItem('maxLineLength', maxLineLength)
        update()
    })

    async function update() {
        root.style.setProperty('--line-length', maxLineLength + 'ch')
        const response = await fetch('/pretty', {
            method: "POST",
            body: JSON.stringify({
                code,
                maxLineLength
            })
		})
		output.innerText = await response.text()
    }
</script>
</html>
`

type Request struct {
	Code          string `json:"code"`
	MaxLineLength int    `json:"maxLineLength"`
}

var portFlag = flag.Int("port", 9090, "port")

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(page))
	})

	http.HandleFunc("/pretty", func(w http.ResponseWriter, r *http.Request) {
		var req Request

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, _ = w.Write([]byte(pretty(req.Code, req.MaxLineLength)))
	})

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *portFlag))
	if err != nil {
		panic(err)
	}
	log.Printf("Listening on http://%s/", ln.Addr().String())
	var srv http.Server
	_ = srv.Serve(ln)
}
