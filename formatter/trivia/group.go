package trivia

// Group partitions a slice of comments into CommentGroups.
// Adjacent comments separated only by whitespace (no blank lines, no code)
// form a single group. A blank line or any non-whitespace content between
// comments starts a new group.
func Group(comments []Comment, source []byte) []*CommentGroup {
	if len(comments) == 0 {
		return nil
	}

	groups := make([]*CommentGroup, 0, 1)
	current := &CommentGroup{
		Comments: []Comment{comments[0]},
	}

	for i := 1; i < len(comments); i++ {
		prev := comments[i-1]
		curr := comments[i]

		if commentsSeparated(prev, curr, source) {
			groups = append(groups, current)
			current = &CommentGroup{
				Comments: []Comment{curr},
			}
		} else {
			current.Comments = append(current.Comments, curr)
		}
	}

	groups = append(groups, current)
	return groups
}

// commentsSeparated returns true if there is a blank line, non-whitespace
// content between two comments, or the first comment is an end-of-line
// comment (shares its line with code).
func commentsSeparated(a, b Comment, source []byte) bool {
	// Blank line (line gap > 1) always separates
	if b.Start.Line-a.End.Line > 1 {
		return true
	}

	// Check for non-whitespace between the comments (code in between)
	startOff := a.End.Offset + 1
	endOff := b.Start.Offset
	if startOff < endOff && startOff < len(source) {
		if endOff > len(source) {
			endOff = len(source)
		}
		for _, c := range source[startOff:endOff] {
			if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
				return true
			}
		}
	}

	// End-of-line comments (code before the comment on the same line)
	// are never grouped with comments on subsequent lines
	if b.Start.Line > a.End.Line && hasCodeBefore(a, source) {
		return true
	}

	return false
}

// hasCodeBefore returns true if there is non-whitespace content before
// the comment on the same line (making it an end-of-line comment).
func hasCodeBefore(c Comment, source []byte) bool {
	lineStart := c.Start.Offset
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	for i := lineStart; i < c.Start.Offset; i++ {
		if source[i] != ' ' && source[i] != '\t' {
			return true
		}
	}
	return false
}
