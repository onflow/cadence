package format

import (
	"bytes"
	"fmt"

	"github.com/onflow/cadence/parser"
	"github.com/turbolent/prettier"
)

// Format parses Cadence source and returns deterministically formatted output.
// filename is used for diagnostics only; the file need not exist on disk.
func Format(src []byte, filename string, opts Options) ([]byte, error) {
	program, err := parser.ParseProgram(nil, src, parser.Config{})
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	doc := program.Doc()

	indent := opts.Indent
	if opts.UseTabs {
		indent = "\t"
	}

	var buf bytes.Buffer
	prettier.Prettier(&buf, doc, opts.LineWidth, indent)

	result := buf.Bytes()

	// Ensure trailing newline
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}

	return result, nil
}
