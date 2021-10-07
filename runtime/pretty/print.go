/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package pretty

import (
	"fmt"
	"io"
	goRuntime "runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/logrusorgru/aurora"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

func colorizeError(message string) string {
	return aurora.Colorize(message, aurora.RedFg|aurora.BrightFg|aurora.BoldFm).String()
}

func colorizeNote(message string) string {
	return aurora.Colorize(message, aurora.CyanFg|aurora.BoldFm).String()
}

func colorizeMessage(message string) string {
	return aurora.Bold(message).String()
}

func colorizeMeta(meta string) string {
	return aurora.Blue(meta).String()
}

const errorPrefix = "error"
const excerptArrow = "--> "
const excerptDots = "... "
const maxLineLength = 500

func FormatErrorMessage(message string, useColor bool) string {
	// prepare prefix
	formattedErrorPrefix := errorPrefix
	if useColor {
		formattedErrorPrefix = colorizeError(errorPrefix)
	}

	// prepare message
	message = ": " + message
	if useColor {
		message = colorizeMessage(message)
	}

	return formattedErrorPrefix + message + "\n"
}

type excerpt struct {
	startPos *ast.Position
	endPos   *ast.Position
	message  string
	isError  bool
}

func newExcerpt(obj interface{}, message string, isError bool) excerpt {
	excerpt := excerpt{
		message: message,
		isError: isError,
	}
	if positioned, hasPosition := obj.(ast.HasPosition); hasPosition {
		startPos := positioned.StartPosition()
		excerpt.startPos = &startPos

		endPos := positioned.EndPosition()
		excerpt.endPos = &endPos
	}
	return excerpt
}

func sortExcerpts(excerpts []excerpt) {
	sort.Slice(excerpts, func(i, j int) bool {
		first := excerpts[i]
		second := excerpts[j]
		if first.startPos == nil || second.startPos == nil {
			return false
		}
		if first.startPos.Line < second.startPos.Line {
			return true
		}
		if first.startPos.Line > second.startPos.Line {
			return false
		}
		if first.startPos.Column < second.startPos.Column {
			return true
		}
		return false
	})
}

type ErrorPrettyPrinter struct {
	writer   io.Writer
	useColor bool
}

func NewErrorPrettyPrinter(writer io.Writer, useColor bool) ErrorPrettyPrinter {
	return ErrorPrettyPrinter{
		writer:   writer,
		useColor: useColor,
	}
}

func (p ErrorPrettyPrinter) writeString(str string) {
	_, err := p.writer.Write([]byte(str))
	if err != nil {
		panic(err)
	}
}

func (p ErrorPrettyPrinter) PrettyPrintError(err error, location common.Location, codes map[common.LocationID]string) error {

	// writeString panics when the write to the writer fails, so recover those errors and return them.
	// This way we don't need to if-err for every single writer write

	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case goRuntime.Error:
				// Don't recover Go's or external panics
				panic(r)
			case error:
				err = r
			default:
				err = fmt.Errorf("%s", r)
			}
		}
	}()

	i := 0
	var printError func(err error, location common.Location) error
	printError = func(err error, location common.Location) error {

		if err, ok := err.(common.HasImportLocation); ok {
			importLocation := err.ImportLocation()
			if importLocation != nil {
				location = importLocation
			}
		}

		if err, ok := err.(errors.ParentError); ok {

			for _, childErr := range err.ChildErrors() {

				childLocation := location

				if childErr, ok := childErr.(common.HasImportLocation); ok {
					importLocation := childErr.ImportLocation()
					if importLocation != nil {
						childLocation = importLocation
					}
				}

				printErr := printError(childErr, childLocation)
				if printErr != nil {
					return printErr
				}
			}

			return nil
		}

		if i > 0 {
			p.writeString("\n")
		}

		var locationID common.LocationID
		if location != nil {
			locationID = location.ID()
		}

		p.prettyPrintError(err, location, codes[locationID])
		i++
		return nil
	}

	return printError(err, location)
}

func (p ErrorPrettyPrinter) prettyPrintError(err error, location common.Location, code string) {

	p.writeString(FormatErrorMessage(err.Error(), p.useColor))

	message := ""
	if secondaryError, ok := err.(errors.SecondaryError); ok {
		message = secondaryError.SecondaryError()
	}

	excerpts := []excerpt{
		newExcerpt(err, message, true),
	}

	if errorNotes, ok := err.(errors.ErrorNotes); ok {
		for _, errorNote := range errorNotes.ErrorNotes() {
			excerpts = append(excerpts,
				newExcerpt(errorNote, errorNote.Message(), false),
			)
		}
	}

	sortExcerpts(excerpts)

	p.writeCodeExcerpts(excerpts, location, code)
}

func (p ErrorPrettyPrinter) writeCodeExcerpts(
	excerpts []excerpt,
	location common.Location,
	code string,
) {
	var lastLineNumber int

	lines := strings.Split(code, "\n")

	for i, excerpt := range excerpts {

		lineNumberString := ""
		lineNumberLength := 0
		if excerpt.startPos != nil {

			plainLineNumberString := strconv.Itoa(excerpt.startPos.Line)
			lineNumberLength = len(plainLineNumberString)

			// prepare line number string
			lineNumberString = plainLineNumberString + " | "
			if p.useColor {
				lineNumberString = colorizeMeta(lineNumberString)
			}
		}

		// write arrow, location, and position (if any)
		if i == 0 {
			p.writeCodeExcerptLocation(location, lineNumberLength, excerpt.startPos)
		}

		// code, if position
		if excerpt.startPos != nil &&
			excerpt.startPos.Line > 0 &&
			excerpt.startPos.Line <= len(lines) &&
			len(code) > 0 {

			if i > 0 && lastLineNumber != 0 && excerpt.startPos.Line-1 > lastLineNumber {
				p.writeCodeExcerptContinuation(lineNumberLength)
			}
			lastLineNumber = excerpt.startPos.Line

			// prepare empty line numbers
			emptyLineNumbers := strings.Repeat(" ", lineNumberLength+1) + "|"
			if p.useColor {
				emptyLineNumbers = colorizeMeta(emptyLineNumbers)
			}

			// empty line
			p.writeString(emptyLineNumbers)
			p.writeString("\n")

			// line number
			p.writeString(lineNumberString)

			// code line
			line := lines[excerpt.startPos.Line-1]
			if len(line) > maxLineLength {
				p.writeString(line[:maxLineLength])
				p.writeString(excerptDots)
			} else {
				p.writeString(line)
			}

			p.writeString("\n")

			// indicator line
			p.writeString(emptyLineNumbers)

			for i := 0; i <= excerpt.startPos.Column; i++ {
				p.writeString(" ")
			}

			columns := 1
			if excerpt.endPos != nil && excerpt.endPos.Line == excerpt.startPos.Line {
				endColumn := excerpt.endPos.Column
				if endColumn >= maxLineLength {
					endColumn = maxLineLength - 1
				}
				columns = endColumn - excerpt.startPos.Column + 1
			}

			indicator := "-"
			if excerpt.isError {
				indicator = "^"
			}

			indicators := strings.Repeat(indicator, columns)
			if p.useColor {
				if excerpt.isError {
					indicators = colorizeError(indicators)
				} else {
					indicators = colorizeNote(indicators)
				}
			}
			p.writeString(indicators)

			if excerpt.message != "" {
				message := excerpt.message
				p.writeString(" ")
				if p.useColor {
					if excerpt.isError {
						message = colorizeError(message)
					} else {
						message = colorizeNote(message)
					}
				}
				p.writeString(message)
			}

			p.writeString("\n")
		} else {
			lastLineNumber = 0
		}
	}
}

func (p ErrorPrettyPrinter) writeCodeExcerptLocation(
	location common.Location,
	lineNumberLength int,
	startPosition *ast.Position,
) {
	// write spaces before arrow
	for i := 0; i < lineNumberLength; i++ {
		p.writeString(" ")
	}

	// write arrow
	if p.useColor {
		p.writeString(colorizeMeta(excerptArrow))
	} else {
		p.writeString(excerptArrow)
	}

	// write location, if any
	if location != nil {
		p.writeString(location.String())
	}

	// write position (line and column)
	if startPosition != nil {
		_, err := fmt.Fprintf(p.writer, ":%d:%d", startPosition.Line, startPosition.Column)
		if err != nil {
			panic(err)
		}
	}
	p.writeString("\n")
}

func (p ErrorPrettyPrinter) writeCodeExcerptContinuation(lineNumberLength int) {
	// write spaces before dots
	for i := 0; i < lineNumberLength; i++ {
		p.writeString(" ")
	}

	// write dots
	dots := excerptDots
	if p.useColor {
		dots = colorizeMeta(dots)
	}
	p.writeString(dots)

	p.writeString("\n")
}
