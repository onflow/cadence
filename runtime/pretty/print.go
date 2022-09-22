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

type Writer interface {
	io.Writer
	io.StringWriter
}

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

const ErrorPrefix = "error"
const messageSeparator = ": "
const excerptArrow = "--> "
const excerptDots = "... "
const maxLineLength = 500

func FormatErrorMessage(prefix string, message string, useColor bool) string {
	if prefix == "" && message == "" {
		return ""
	}

	var builder strings.Builder

	if useColor {
		builder.WriteString(colorizeError(prefix))
		builder.WriteString(colorizeMessage(messageSeparator))
		builder.WriteString(colorizeMessage(message))
	} else {
		builder.WriteString(prefix)
		builder.WriteString(messageSeparator)
		builder.WriteString(message)
	}

	builder.WriteByte('\n')

	return builder.String()
}

type excerpt struct {
	startPos *ast.Position
	endPos   *ast.Position
	message  string
	isError  bool
}

func newExcerpt(obj any, message string, isError bool) excerpt {
	excerpt := excerpt{
		message: message,
		isError: isError,
	}
	if positioned, hasPosition := obj.(ast.HasPosition); hasPosition {
		startPos := positioned.StartPosition()
		excerpt.startPos = &startPos

		endPos := positioned.EndPosition(nil)
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
	writer   Writer
	useColor bool
}

func NewErrorPrettyPrinter(writer Writer, useColor bool) ErrorPrettyPrinter {
	return ErrorPrettyPrinter{
		writer:   writer,
		useColor: useColor,
	}
}

func (p ErrorPrettyPrinter) writeString(str string) {
	_, err := p.writer.WriteString(str)
	if err != nil {
		panic(err)
	}
}

func (p ErrorPrettyPrinter) PrettyPrintError(
	err error,
	location common.Location,
	codes map[common.Location][]byte,
) error {

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

		if err, ok := err.(common.HasLocation); ok {
			importLocation := err.ImportLocation()
			if importLocation != nil {
				location = importLocation
			}
		}

		if err, ok := err.(errors.ParentError); ok {

			for _, childErr := range err.ChildErrors() {

				childLocation := location

				if childErr, ok := childErr.(common.HasLocation); ok {
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

		p.prettyPrintError(err, location, codes[location])
		i++
		return nil
	}

	return printError(err, location)
}

func (p ErrorPrettyPrinter) prettyPrintError(err error, location common.Location, code []byte) {

	prefix := ErrorPrefix
	if secondaryError, ok := err.(errors.HasPrefix); ok {
		prefix = secondaryError.Prefix()
	}

	p.writeString(FormatErrorMessage(prefix, err.Error(), p.useColor))

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
	code []byte,
) {
	var lastLineNumber int

	codeLines := strings.Split(string(code), "\n")

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

		haveCode := excerpt.startPos != nil &&
			excerpt.startPos.Line > 0 &&
			excerpt.startPos.Line <= len(codeLines) &&
			len(code) > 0

		// if we do not have code or code position skip further processing of this excerpt
		if !haveCode {
			lastLineNumber = 0
			continue
		}

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
		codeLine := codeLines[excerpt.startPos.Line-1]
		if len(codeLine) > maxLineLength {
			p.writeString(codeLine[:maxLineLength])
			p.writeString(excerptDots)
		} else {
			p.writeString(codeLine)
		}

		p.writeString("\n")

		// indicator line
		p.writeString(emptyLineNumbers)

		indicatorLength := excerpt.startPos.Column
		if indicatorLength >= maxLineLength {
			indicatorLength = maxLineLength
		}

		p.writeString(" ")

		// write the whitespace prefix to align the indicator
		if codeLine != "" {

			// determine how many characters to insert.
			limit := indicatorLength
			if len(codeLine) < limit {
				limit = len(codeLine)
			}

			for i := 0; i < limit; i++ {
				c := codeLine[i]
				if c != '\t' {
					c = ' '
				}
				p.writeString(string(c))
			}

		} else {
			s := strings.Repeat(" ", indicatorLength)
			p.writeString(s)
		}

		// determine how many times we need to repeat the indicator character
		width := 1
		if excerpt.endPos != nil && excerpt.endPos.Line == excerpt.startPos.Line {
			endColumn := excerpt.endPos.Column
			if endColumn >= maxLineLength {
				endColumn = maxLineLength - 1
			}
			width = endColumn - excerpt.startPos.Column + 1
		}

		// determine which indicator to use - if this is the erring line,
		// use the upper arrow to point to the faulty code;
		// if this is not the error indicator, don't repeat it
		indicator := "-"
		if excerpt.isError {
			indicator = strings.Repeat("^", width)
		}

		if p.useColor {
			if excerpt.isError {
				indicator = colorizeError(indicator)
			} else {
				indicator = colorizeNote(indicator)
			}
		}
		p.writeString(indicator)

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
		s := fmt.Sprintf(":%d:%d", startPosition.Line, startPosition.Column)
		p.writeString(s)
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
