/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// wrapperCode is the Go code that is temporarily wrapped around the expression
// to parse the expression as a file.
//
// NOTE: also adjust the exprRange below if you change this
const wrapperCode = `
package p

func main(){
var e = %s
}
`

// exprRange is the range of '%s' in wrapperCode,
// in terms of offsets from left and right.
//
// NOTE: depends on wrapperCode
var exprRange = struct{ left, right int }{left: 34, right: 3}

var flagChanges = flag.String("changes", "", "file containing changes to apply to the expression")
var flagCode = flag.String("code", "", "file containing the Go expression to patch")
var flagMCP = flag.Bool("mcp", false, "run as MCP server")
var flagWeb = flag.Bool("web", false, "run as web server")
var portFlag = flag.Int("port", 9090, "port")

func main() {
	flag.Parse()

	if *flagMCP {
		runMCP()
	} else if *flagWeb {
		runWeb()
	} else {
		// Read code

		if *flagCode == "" {
			panic("flag -code is required")
		}

		code, err := os.ReadFile(*flagCode)
		if err != nil {
			panic(fmt.Sprintf("failed to read code file %q: %v", *flagCode, err))
		}

		// Read changes

		if *flagChanges == "" {
			panic("flag -changes is required")
		}

		var changes [][]byte

		changesData, err := os.ReadFile(*flagChanges)
		if err != nil {
			panic(fmt.Sprintf("failed to read changes file %q: %v", *flagChanges, err))
		}

		for _, change := range bytes.Split(changesData, []byte("\n")) {
			change = bytes.TrimSpace(change)
			if len(change) == 0 {
				continue
			}
			changes = append(changes, change)
		}

		result := patchCode(code, changes)
		fmt.Println(string(result))
	}
}

const changeDescription = `Each change has the format 'path: oldValue != newValue',

where 'path' is a sequence of one of the following:

- '[index]': for accessing an element of an array
- '.name': for accessing a field in a composite literal

For example, a change '.x.ys[0]: 1 != 2'
applied to the expression 'Outer{x: Inner{ys: []int{1}}}'
returns the expression 'Outer{x: Inner{ys: []int{2}}}'
`

const commandDescription = `Apply a list of changes to a Go expression. ` + changeDescription

// patchCode applies the given changes to the provided code.
// For the change format, see the `applyChange` function.
func patchCode(code []byte, changes [][]byte) []byte {
	if !bytes.HasSuffix(code, []byte("\n")) {
		code = append(code, '\n')
	}

	f, err := decorator.Parse(
		fmt.Sprintf(
			wrapperCode,
			string(code),
		),
	)
	if err != nil {
		panic(err)
	}

	// func main() { ... }
	wrapperStmt := f.Decls[0].(*dst.FuncDecl).Body.List[0]
	// var e = ...
	e := wrapperStmt.(*dst.DeclStmt).Decl.(*dst.GenDecl).Specs[0].(*dst.ValueSpec).Values[0]

	for _, change := range changes {
		applyChange(change, e)
	}

	var res bytes.Buffer
	if err := decorator.Fprint(&res, f); err != nil {
		panic(err)
	}

	code = res.Bytes()
	code = code[exprRange.left : len(code)-exprRange.right]

	return code
}

// applyChange applies a change to the given expression `v`.
// The change is expected to be in the format:
//
//	`path: oldValue != newValue`
//
// where `path` is a sequence of one of the following:
// - `[index]` for accessing an element of an array
// - `.name` for accessing a field in a composite literal
func applyChange(change []byte, v dst.Expr) {

	change = bytes.TrimSpace(change)

	for {
		var op byte
		op, change = change[0], change[1:]

		switch op {
		case '[':
			endOffset := bytes.IndexByte(change, ']')
			indexStr := string(change[:endOffset])
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				panic(fmt.Sprintf("invalid index in change: %s", indexStr))
			}
			// +1 to skip the ']'
			change = change[endOffset+1:]

			v = v.(*dst.CompositeLit).Elts[index]

		case '.':
			endOffset := bytes.IndexFunc(change, func(r rune) bool {
				return !isIdentifierChar(r)
			})
			name := string(change[:endOffset])
			change = change[endOffset:]

			if unaryExpr, ok := v.(*dst.UnaryExpr); ok {
				v = unaryExpr.X
			}

			var found bool

			for _, e := range v.(*dst.CompositeLit).Elts {
				if kv, ok := e.(*dst.KeyValueExpr); ok && kv.Key.(*dst.Ident).Name == name {
					v = kv.Value
					found = true
					break
				}
			}

			if !found {
				panic(fmt.Sprintf("could not find key %q in composite literal: %#+v", name, v))
			}

		case ':':
			parts := bytes.Split(change, []byte(" != "))
			if len(parts) != 2 {
				panic(fmt.Sprintf("invalid change format, expected 'path: old != new', got: %s", change))
			}

			changeOld, changeNew := bytes.TrimSpace(parts[0]), bytes.TrimSpace(parts[1])

			current := v.(*dst.BasicLit).Value
			if current != string(changeOld) {
				panic(fmt.Sprintf("expected old value %q, got %q", changeOld, current))
			}

			v.(*dst.BasicLit).Value = string(changeNew)

			return

		default:
			panic(fmt.Sprintf("unexpected character in change: %c", op))
		}
	}
}

func isIdentifierChar(r rune) bool {
	return r == '_' ||
		(r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

func runMCP() {
	// Create a server with a single tool.
	s := server.NewMCPServer(
		"go-apply-expr-diff",
		"v1.0.0",
	)

	type params struct {
		ExpressionCode string   `json:"expression_code" jsonschema:"Go expression code"`
		Changes        []string `json:"changes" jsonschema:"changes to apply to the expression"`
	}

	s.AddTool(
		mcp.NewTool("applyExpressionCodeChanges",
			mcp.WithDescription(commandDescription),
			mcp.WithString(
				"code",
				mcp.Required(),
				mcp.Description("Go expression code"),
			),
			mcp.WithArray(
				"changes",
				mcp.Required(),
				mcp.Description(changeDescription),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (res *mcp.CallToolResult, err error) {
			defer func() {
				if err != nil {
					res = mcp.NewToolResultError(err.Error())
					err = nil
				}
			}()

			codeArgument, err := request.RequireString("code")
			if err != nil {
				return nil, fmt.Errorf("missing or invalid 'code' parameter: %w", err)
			}

			changesArgument, err := request.RequireStringSlice("changes")
			if err != nil {
				return nil, fmt.Errorf("missing or invalid 'changes' parameter: %w", err)
			}

			changes := make([][]byte, 0, len(changesArgument))

			for _, change := range changesArgument {
				changes = append(changes, []byte(change))
			}

			code := []byte(codeArgument)

			result := patchCode(code, changes)

			return mcp.NewToolResultText(string(result)), nil
		},
	)

	err := server.ServeStdio(s)
	if err != nil {
		panic(err)
	}
}

// language=html
const page = `
<html>
<head>
    <title>Patch</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: monospace;
            height: 100vh;
        }

        #panels {
            display: grid;
            grid-template-rows: 100vh;
            grid-template-columns: repeat(3, 1fr);
            grid-template-areas: "code changes output";
        }

        #code {
            grid-area: code;
            border: 1px solid #ccc;
            resize: none;
            white-space: nowrap;
        }

        #changes {
            grid-area: changes;
            border: 1px solid #ccc;
            resize: none;
            white-space: nowrap;
        }

        #output {
            white-space: pre;
            height: 100%;
            overflow: scroll;
        }

        #output.error {
            color: red;
        }
    </style>
</head>
<body id="panels">
<textarea id="code"></textarea>
<textarea id="changes"></textarea>
<div id="output"></div>
</body>
<script>
    const codeElement = document.getElementById("code")
    const changesElement = document.getElementById("changes")
    const outputElement = document.getElementById("output")

    let code = localStorage.getItem('code') || ''
    let changes = localStorage.getItem('changes') || ''

    document.addEventListener('DOMContentLoaded', () => {
        codeElement.value = code
        changesElement.value = changes
        update()
    })

    codeElement.addEventListener("input", (e) => {
        code = e.target.value
        localStorage.setItem('code', code)
        update()
    })

    changesElement.addEventListener("input", (e) => {
        changes = e.target.value
        localStorage.setItem('changes', changes)
        update()
    })

    async function update() {
        let response
        try {
            response = await fetch('/apply', {
                method: "POST",
                body: JSON.stringify({
                    code,
                    changes
                })
            })

            outputElement.innerText = await response.text()

            if (response.ok) {
                outputElement.classList.remove('error')
            } else {
                outputElement.classList.add('error')
            }
        } catch (e) {
            outputElement.classList.add('error')
            outputElement.innerText = 'Error: ' + e.message
        }
    }
</script>
</html>
`

func runWeb() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(page))
	})

	http.HandleFunc("/apply", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("Error: %v", err)
				http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
				return
			}
		}()

		var req struct {
			Code    string `json:"code"`
			Changes string `json:"changes"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var changes [][]byte

		for _, change := range strings.Split(req.Changes, "\n") {
			change = strings.TrimSpace(change)
			if len(change) == 0 {
				continue
			}
			changes = append(changes, []byte(change))
		}

		_, _ = w.Write(patchCode([]byte(req.Code), changes))
	})

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *portFlag))
	if err != nil {
		panic(err)
	}

	log.Printf("Listening on http://%s/", ln.Addr().String())
	var srv http.Server
	_ = srv.Serve(ln)
}
