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

// Read a (location, code) CSV from stdin and write it to stdout,
// replacing the code of the row matching the given location with
// the contents of the given file. The first row is preserved as-is.
package main

import (
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
)

var locationFlag = flag.String("location", "", "location whose code should be replaced")
var fileFlag = flag.String("file", "", "path to file containing the new code")

func main() {
	flag.Parse()

	if *locationFlag == "" || *fileFlag == "" {
		flag.Usage()
		os.Exit(2)
	}

	newCode, err := os.ReadFile(*fileFlag)
	if err != nil {
		log.Fatalf("failed to read %s: %s", *fileFlag, err)
	}

	r := csv.NewReader(os.Stdin)
	r.FieldsPerRecord = -1

	w := csv.NewWriter(os.Stdout)

	first := true
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read CSV: %s", err)
		}

		if !first && len(row) >= 2 && row[0] == *locationFlag {
			row[1] = string(newCode)
		}
		first = false

		if err := w.Write(row); err != nil {
			log.Fatalf("failed to write CSV: %s", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatalf("failed to flush CSV: %s", err)
	}
}
