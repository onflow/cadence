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
	"flag"
	"io"
	"log"
	"os"
	"strings"
)

const commentsPrefix = "//"

// A minifier to minify a Cadence script file. Currently, it only removes comments and new lines.
// Usage: go run minifier.go -i inputfile.cdc -o outputfile.cdc
// e.g. go run minifier.go -i ../../../transactions/transfer_tokens.cdc -o /tmp/test.cdc
func main() {
	inputFile := flag.String("i", "", "the cadence file to minify")
	outputFile := flag.String("o", "", "the output file")
	flag.Parse()

	if *inputFile == "" {
		log.Fatal("input file not provided")
	}

	if *outputFile == "" {
		log.Fatal("output file not provided")
	}

	log.Println("input file:", *inputFile)
	log.Println("output file:", *outputFile)

	err := minify(*inputFile, *outputFile)
	if err != nil {
		log.Fatalf("failed to minify %s", *inputFile)
	}

	log.Println("done")
}

func minify(inputFile, outputFile string) error {
	input, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer output.Close()

	reader := bufio.NewReader(input)
	writer := bufio.NewWriter(output)

	eof := false
	for {

		if eof {
			return nil
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// we have reached the eof but need to still process the last line
				eof = true
			} else {
				return err
			}
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, commentsPrefix) {
			continue
		}
		_, err = writer.WriteString(line)
		if err != nil {
			return err
		}

		if !eof {
			_, err = writer.WriteRune('\n')
			if err != nil {
				return err
			}
		}

		err = writer.Flush()
		if err != nil {
			return err
		}
	}
}
