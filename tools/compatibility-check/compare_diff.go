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

package compatibility_check

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kylelemons/godebug/diff"
)

func CompareFiles(oldResults, newResults *os.File) error {

	oldResultsScanner := bufio.NewScanner(oldResults)
	oldResultsScanner.Split(bufio.ScanLines)
	var oldResultsLines []string
	for oldResultsScanner.Scan() {
		oldResultsLines = append(oldResultsLines, oldResultsScanner.Text())
	}

	newResultsScanner := bufio.NewScanner(newResults)
	newResultsScanner.Split(bufio.ScanLines)
	var newResultsLines []string
	for newResultsScanner.Scan() {
		newResultsLines = append(newResultsLines, newResultsScanner.Text())
	}

	diffChunks := diff.DiffChunks(oldResultsLines, newResultsLines)

	totalDiffs := 0
	for i, currentChunk := range diffChunks {
		if currentChunk.Added == nil && currentChunk.Deleted == nil {
			continue
		}

		prevChunk := diffChunks[i-1]
		printDiffChunk(prevChunk, currentChunk)

		totalDiffs += 1
	}

	if totalDiffs > 0 {
		return errors.New("found differences")
	}

	return nil
}

func printDiffChunk(prevChunk, currentChunk diff.Chunk) {
	sb := strings.Builder{}

	var extraLinesToPrint = 4

	// Print the last few lines from the previous chunk.
	equalLines := prevChunk.Equal
	equalLinesLen := len(equalLines)
	if equalLinesLen <= extraLinesToPrint {
		extraLinesToPrint = equalLinesLen
	}

	printFrom := equalLinesLen - extraLinesToPrint
	for i := printFrom; i < equalLinesLen; i++ {
		// Thus, Print the previous lines.
		sb.WriteString(equalLines[i])
		sb.WriteRune('\n')
	}

	// Print additions
	for _, line := range currentChunk.Added {
		sb.WriteString("+")
		sb.WriteString(line)
	}

	// Print deletions
	for _, line := range currentChunk.Deleted {
		sb.WriteString("-")
		sb.WriteString(line)
	}

	// If there are 'Equal' lines, that means this is the last chunk of a diff.
	if len(currentChunk.Equal) > 0 {
		// Keep some space between this diff and the next diff.
		sb.WriteRune('\n')
	}

	fmt.Println(sb.String())
}
