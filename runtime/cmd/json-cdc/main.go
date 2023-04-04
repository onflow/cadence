/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/k0kubun/pp/v3"

	jsoncdc "github.com/onflow/cadence/encoding/json"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintf(os.Stderr, "expected command\n")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "decode":
		var data bytes.Buffer
		reader := bufio.NewReader(os.Stdin)
		_, err := io.Copy(&data, reader)
		if err != nil {
			panic(err)
		}

		value, err := jsoncdc.Decode(nil, data.Bytes())
		if err != nil {
			panic(err)
		}

		_, _ = pp.Print(value)

	default:
		_, _ = fmt.Fprintf(os.Stderr, "unsupported command: %s", command)
		os.Exit(1)
	}
}
