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

	// Print the last few line from the previous chunk.
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
