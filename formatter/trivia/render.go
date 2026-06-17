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

package trivia

import (
	"strings"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/ast"
)

// CommentMap implements ast.PrettyContext. These methods are the bridge
// between the comment data captured by Attach() and the rendering pass
// performed by ast.*.Doc(ctx).

var _ ast.PrettyContext = (*CommentMap)(nil)

// Wrap wraps an element's structural doc with its leading, same-line, and
// trailing comments, plus a trailing semicolon if the element had one in
// the source. Comments are consumed (removed) on first call so each comment
// is emitted exactly once.
//
// Separator choice for leading comments depends on the group:
//   - A line comment (`//`) terminates its line, so it requires HardLine after.
//   - A block comment (`/* */`) on the same source line as the element renders
//     inline with a Space separator (preserving `/* note */ code`).
//   - A block comment on a different line than the element renders with HardLine
//     so it stays on its own line.
func (cm *CommentMap) Wrap(elem ast.Element, doc prettier.Doc) prettier.Doc {
	leading, sameLine, trailing := cm.TakeRaw(elem)
	hasSemi := cm.Semicolons[elem]

	if len(leading) == 0 && sameLine == nil && len(trailing) == 0 && !hasSemi {
		return doc
	}

	parts := prettier.Concat{}

	elemStartLine := elem.StartPosition().Line
	for _, g := range leading {
		parts = append(parts, renderCommentGroup(g))
		if leadingShouldStayInline(g, elemStartLine) {
			parts = append(parts, prettier.Text(" "))
		} else {
			parts = append(parts, prettier.HardLine{})
		}
	}

	parts = append(parts, doc)

	if hasSemi {
		parts = append(parts, prettier.Text(";"))
	}

	if sameLine != nil {
		parts = append(parts, prettier.Text("  "), renderCommentGroup(sameLine))
	}

	for _, g := range trailing {
		parts = append(parts, prettier.HardLine{}, renderCommentGroup(g))
	}

	return parts
}

// leadingShouldStayInline reports whether a leading comment group should
// render inline (with a Space separator) rather than on its own line. True
// only when every comment in the group is a block comment AND the group's
// last comment ends on the same source line as the element starts.
func leadingShouldStayInline(g *CommentGroup, elemStartLine int) bool {
	if len(g.Comments) == 0 {
		return false
	}
	for _, c := range g.Comments {
		if c.Kind == KindLine || c.Kind == KindDocLine {
			return false
		}
	}
	last := g.Comments[len(g.Comments)-1]
	return last.End.Line == elemStartLine
}

// Take consumes elem's leading, same-line, and trailing comments and returns
// them as already-rendered prettier docs. Returns nil for each slot that has
// no comments. After Take, calling Wrap on the same elem returns the doc
// unchanged for the comment parts (but still applies semicolon).
func (cm *CommentMap) Take(elem ast.Element) (leading, sameLine, trailing prettier.Doc) {
	leadGroups, sameGroup, trailGroups := cm.TakeRaw(elem)

	if len(leadGroups) > 0 {
		parts := prettier.Concat{}
		for i, g := range leadGroups {
			if i > 0 {
				parts = append(parts, prettier.HardLine{})
			}
			parts = append(parts, renderCommentGroup(g))
		}
		leading = parts
	}

	if sameGroup != nil {
		sameLine = renderCommentGroup(sameGroup)
	}

	if len(trailGroups) > 0 {
		parts := prettier.Concat{}
		for i, g := range trailGroups {
			if i > 0 {
				parts = append(parts, prettier.HardLine{})
			}
			parts = append(parts, renderCommentGroup(g))
		}
		trailing = parts
	}

	return
}

// HasComments reports whether elem has any attached comments.
func (cm *CommentMap) HasComments(elem ast.Element) bool {
	return len(cm.Leading[elem]) > 0 ||
		cm.SameLine[elem] != nil ||
		len(cm.Trailing[elem]) > 0
}

// HasLeadingLineComment reports whether elem has a leading `//`-style line comment.
// Block comments (`/* */`) don't count — they don't terminate the rest of the line
// when rendered inline, so they don't force a line break.

// BlankLineBetween reports whether the source had a blank line between
// the two elements. Uses byte-level scanning of cm.Source between the prev
// element's end (including its trailing comments) and the next element's
// start (including its leading comments).
func (cm *CommentMap) BlankLineBetween(prev, next ast.Element) bool {
	if len(cm.Source) == 0 || prev == nil || next == nil {
		return false
	}

	endOffset := prev.EndPosition(nil).Offset
	if trailing := cm.Trailing[prev]; len(trailing) > 0 {
		if tEnd := trailing[len(trailing)-1].EndPos().Offset; tEnd > endOffset {
			endOffset = tEnd
		}
	}

	startOffset := next.StartPosition().Offset
	if leading := cm.Leading[next]; len(leading) > 0 {
		if lStart := leading[0].StartPos().Offset; lStart < startOffset {
			startOffset = lStart
		}
	}

	if endOffset >= startOffset || endOffset >= len(cm.Source) {
		return false
	}

	sawNewline := false
	for i := endOffset; i < startOffset && i < len(cm.Source); i++ {
		b := cm.Source[i]
		if b == '\n' {
			if sawNewline {
				return true
			}
			sawNewline = true
		} else if b != ' ' && b != '\t' && b != '\r' {
			sawNewline = false
		}
	}
	return false
}

// Header returns header comments (above first declaration) as a prettier doc,
// followed by HardLines and a blank-line separator. Returns nil if no header.
// Header is consumed on first call.
func (cm *CommentMap) Header() prettier.Doc {
	header := cm.HeaderComments
	cm.HeaderComments = nil
	if len(header) == 0 {
		return nil
	}
	parts := prettier.Concat{}
	for _, g := range header {
		parts = append(parts, renderCommentGroup(g), prettier.HardLine{})
	}
	// Blank line between header and first declaration.
	parts = append(parts, prettier.HardLine{})
	return parts
}

// Footer returns footer comments (below last declaration) as a prettier doc,
// preceded by a blank-line separator. Returns nil if no footer.
// Footer is consumed on first call.
func (cm *CommentMap) Footer() prettier.Doc {
	footer := cm.FooterComments
	cm.FooterComments = nil
	if len(footer) == 0 {
		return nil
	}
	parts := prettier.Concat{prettier.HardLine{}}
	for _, g := range footer {
		parts = append(parts, prettier.HardLine{}, renderCommentGroup(g))
	}
	return parts
}

// renderCommentGroup renders a group of comments, each on its own line.
func renderCommentGroup(g *CommentGroup) prettier.Doc {
	if len(g.Comments) == 1 {
		return renderComment(g.Comments[0])
	}

	parts := prettier.Concat{}
	for i, c := range g.Comments {
		if i > 0 {
			parts = append(parts, prettier.HardLine{})
		}
		parts = append(parts, renderComment(c))
	}
	return parts
}

// renderComment renders a single comment. Line comments have trailing
// whitespace trimmed.
func renderComment(c Comment) prettier.Doc {
	text := c.Text
	switch c.Kind {
	case KindLine, KindDocLine:
		text = strings.TrimRight(text, " \t")
	}
	return prettier.Text(text)
}
