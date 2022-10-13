//go:build wasm
// +build wasm

/*
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
	"fmt"
	"log"
	"runtime/debug"
	"syscall/js"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser"
)

const globalFunctionNamePrefix = "CADENCE_PARSER"

func globalFunctionName(name string) string {
	return fmt.Sprintf("__%s_%s__", globalFunctionNamePrefix, name)
}

func main() {

	log.Println("Cadence Parser")

	done := make(chan struct{}, 0)

	js.Global().Set(
		globalFunctionName("parse"),
		js.FuncOf(func(this js.Value, args []js.Value) any {
			code := args[0].String()
			return parse(code)
		}),
	)
	<-done
}

type result struct {
	Program *ast.Program `json:"program"`
	Error   error        `json:"error,omitempty"`
}

func parse(code string) string {

	var res result

	func() {
		defer func() {
			if r := recover(); r != nil {
				res.Error = fmt.Errorf("%s", debug.Stack())
			}
		}()

		res.Program, res.Error = parser.ParseProgram([]byte(code), nil)
	}()

	serialized, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}

	return string(serialized)
}
