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
	"encoding/csv"
	"log"
	"net/http"
	"os"

	"github.com/onflow/cadence/errors"
)

type namedError struct {
	name string
	error
}

func main() {
	if len(os.Args) < 1 {
		log.Printf("Usage: %s <write-csv|validate-doc-links>", os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "write-csv":
		writeCSV()
	case "validate-doc-links":
		validateDocLinks()
	}
}

func writeCSV() {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	err := w.Write([]string{"name", "example"})
	if err != nil {
		log.Fatalf("Failed to write CSV header: %v", err)
	}

	for _, namedErr := range generateErrors() {
		err = w.Write([]string{namedErr.name, namedErr.Error()})
		if err != nil {
			log.Fatalf("Failed to write CSV row: %v", err)
		}
	}

}

func validateDocLinks() {

	var failed bool

	for _, namedErr := range generateErrors() {
		hasDocumentationLink, ok := namedErr.error.(errors.HasDocumentationLink)
		if !ok {
			continue
		}

		link := hasDocumentationLink.DocumentationLink()
		if link == placeholderString {
			continue
		}

		resp, err := http.Head(link)
		if err != nil {
			log.Printf("Error checking documentation link for error %q: %v", namedErr.name, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Invalid documentation link for error %q: %v", namedErr.name, link)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}
