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
	"fmt"
	"log"

	"encoding/json"
	"runtime/debug"
	"syscall/js"

	"github.com/onflow/cadence/tools/docgen"
)

const globalFunctionNamePrefix = "CADENCE_DOCGEN"

func globalFunctionName(name string) string {
	return fmt.Sprintf("__%s_%s__", globalFunctionNamePrefix, name)
}

func main() {
	log.Println("Cadence Documentation Generator")

	done := make(chan struct{}, 0)

	js.Global().Set(
		globalFunctionName("generate"),
		js.FuncOf(
			func(this js.Value, args []js.Value) any {
				return generateDocs(args)
			},
		),
	)

	<-done
}

type documentations map[string]string

type result struct {
	Docs  documentations `json:"docs"`
	Error error          `json:"error,omitempty"`
}

func generateDocs(args []js.Value) string {

	var res result

	func() {
		programArgsCount := len(args)
		if programArgsCount < 1 {
			res.Error = fmt.Errorf("not enough arguments: expected 1, found %d", programArgsCount)
			return
		}

		if programArgsCount > 1 {
			res.Error = fmt.Errorf("too many arguments: expected 1, found %d", programArgsCount)
			return
		}

		code := args[0].String()

		defer func() {
			if r := recover(); r != nil {
				res.Error = fmt.Errorf("%s", debug.Stack())
			}
		}()

		docGen := docgen.NewDocGenerator()
		docs, err := docGen.GenerateInMemory(code)

		if err != nil {
			res.Error = err
			return
		}

		// Convert the byte content to string before sending as json.
		res.Docs = documentations{}
		for fileName, content := range docs {
			res.Docs[fileName] = string(content)
		}
	}()

	serialized, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}

	return string(serialized)
}
