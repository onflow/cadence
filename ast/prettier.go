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

package ast

import (
	"strings"

	"github.com/turbolent/prettier"
)

// PrettyContext provides hooks that let a renderer interleave comments,
// preserve blank lines, and attach trailing markers (e.g., semicolons)
// into the document produced by ast.Doc().
//
// NopContext can be used by callers that don't have comment data
// and want a canonical structural document.
type PrettyContext interface {
	// Wrap wraps an element's structural doc with its surrounding comments
	// (leading, same-line, trailing) and any descendant comments that have
	// no natural position in the doc tree. Idempotent within a single render:
	// once an element's comments are consumed, subsequent Wrap calls return
	// the doc unchanged.
	Wrap(elem Element, doc prettier.Doc) prettier.Doc

	// Take consumes and returns rendered docs for elem's leading, same-line,
	// and trailing comments. Used by list-building methods that need to weave
	// comments around separators (commas).
	Take(elem Element) (leading, sameLine, trailing prettier.Doc)

	// HasComments reports whether elem has any attached comments. Used to
	// choose between soft-break and hard-break list layouts.
	HasComments(elem Element) bool

	// HasLeadingLineComment reports whether elem has a leading `//`-style
	// line comment. Used to decide whether to break a line before a value
	// (e.g., between `=` and the assigned value when a leading line comment
	// would otherwise swallow the rest of the line).
	HasLeadingLineComment(elem Element) bool

	// BlankLineBetween reports whether the source had a blank line between
	// two consecutive sibling elements.
	BlankLineBetween(prev, next Element) bool

	// Header returns header comments (above the first declaration), or nil.
	Header() prettier.Doc

	// Footer returns footer comments (below the last declaration), or nil.
	Footer() prettier.Doc
}

// NopContext is a no-op PrettyContext: ast.Doc(NopContext{}) returns the
// canonical structural document with no comments interleaved.
type NopContext struct{}

func (NopContext) Wrap(_ Element, doc prettier.Doc) prettier.Doc {
	return doc
}

func (NopContext) Take(_ Element) (prettier.Doc, prettier.Doc, prettier.Doc) {
	return nil, nil, nil
}

func (NopContext) HasComments(_ Element) bool {
	return false
}

func (NopContext) HasLeadingLineComment(_ Element) bool {
	return false
}

func (NopContext) BlankLineBetween(_, _ Element) bool {
	return false
}

func (NopContext) Header() prettier.Doc {
	return nil
}

func (NopContext) Footer() prettier.Doc {
	return nil
}

type Pretty interface {
	Doc(ctx PrettyContext) prettier.Doc
}

func Prettier(element Pretty) string {
	var builder strings.Builder
	doc := element.Doc(NopContext{}).Flatten()
	prettier.Prettier(&builder, doc, 80, "    ")
	return builder.String()
}

// docOrEmpty returns the document of the given element,
// or an empty text document if the element is the zero value.
//
// NOTE: The function is generic because the version which takes an interface
// will not properly allow for a nil check: When passing a pointer typed value as an interface,
// a typed nil is created, which is not equal to nil.
func docOrEmpty[T interface {
	Pretty
	comparable
}](element T, ctx PrettyContext) prettier.Doc {
	var empty T
	if element == empty {
		return prettier.Text("")
	}
	return element.Doc(ctx)
}
