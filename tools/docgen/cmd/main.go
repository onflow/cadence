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
	"fmt"
	"log"
	"os"

	"io/ioutil"

	"github.com/onflow/cadence/tools/docgen"
)

func main() {
	programArgsCount := len(os.Args) - 1
	if programArgsCount < 2 {
		log.Fatalf("Not enough arguments: expected 2, found %d", programArgsCount)
	}

	if programArgsCount > 2 {
		log.Fatalf("Too many arguments: expected 2, found %d", programArgsCount)
	}

	input := os.Args[1]
	outputDir := os.Args[2]

	content, err := ioutil.ReadFile(input)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo, err := os.Stat(outputDir)

	if os.IsNotExist(err) {
		log.Fatalf("No such file or directory: %s", outputDir)
	}

	if !fileInfo.IsDir() {
		log.Fatalf("Not a directory: %s", outputDir)
	}

	code := string(content)

	docGen := docgen.NewDocGenerator()
	err = docGen.Generate(code, outputDir)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(fmt.Sprintf("Docs generated at: %s", outputDir))
}
