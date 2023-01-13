/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2023 Dapper Labs, Inc.
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
	"os"

	"github.com/rs/zerolog/log"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func main() {
	if len(os.Args) < 2 {
		log.Error().Msg("not enough arguments. Usage: old_checking_results new_checking_results")
		return
	}

	oldResultsPath := os.Args[1]
	newResultsPath := os.Args[2]

	oldResultsFile, err := os.ReadFile(oldResultsPath)
	if err != nil {
		log.Err(err).Msgf("failed to open file: %s", oldResultsPath)
		return
	}

	newResultsFile, err := os.ReadFile(newResultsPath)
	if err != nil {
		log.Err(err).Msgf("failed to open file: %s", newResultsPath)
		return
	}

	compareBytes(oldResultsFile, newResultsFile)
}

func compareBytes(old, new []byte) {
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(string(old), string(new), false)

	changes := make([]diffmatchpatch.Diff, 0)

	// Filter out only the diff chunks with changes.
	// No need to print the equal chunks.
	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffEqual {
			continue
		}
		changes = append(changes, diff)
	}

	fmt.Println(dmp.DiffPrettyText(changes))

	if len(changes) > 0 {
		log.Fatal().Msg("found differences")
	}
}
