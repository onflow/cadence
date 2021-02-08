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

// A utility program that parses a state dump in JSON Lines format and decodes all values

package main

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/onflow/cadence/runtime/interpreter"
)

func main() {
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var entry struct {
		Value string
	}

	var decoded int

	decoder := json.NewDecoder(file)
	for lines := 0; ; lines++ {
		if lines%100 == 0 {
			print(".")
		}

		err = decoder.Decode(&entry)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		var data []byte
		data, err = hex.DecodeString(entry.Value)
		if err != nil {
			log.Fatal(err)
		}

		var version uint16
		data, version = interpreter.StripMagic(data)
		if version == 0 {
			continue
		}
		_, err = interpreter.DecodeValue(data, nil, nil, version, nil)
		if err != nil {
			log.Fatal(err)
		}

		decoded++
	}

	println()

	log.Printf("successfully decoded %d values", decoded)
}
