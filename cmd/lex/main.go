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
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/onflow/cadence/parser/lexer"
)

func main() {

	var path string
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	var (
		data []byte
		err  error
	)
	if len(path) == 0 {
		data, err = io.ReadAll(bufio.NewReader(os.Stdin))
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error reading file %s: %v\n", path, err)
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", debug.Stack())
		}
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}()

	_, _ = fmt.Fprintf(os.Stderr, "lexing %s\n", path)

	tokens, err := lexer.Lex(data, nil)
	if err != nil {
		return
	}
	defer tokens.Reclaim()

	for {
		token := tokens.Next()
		if token.Type == lexer.TokenEOF {
			break
		}

		fmt.Printf("%s-%s: %s\n", token.StartPos, token.EndPos, token.Type)
	}
}
