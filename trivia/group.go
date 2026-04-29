package trivia

// Group partitions a slice of comments into CommentGroups.
// Adjacent comments separated only by whitespace (no blank lines)
// form a single group. A blank line between comments starts a new group.
func Group(comments []Comment) []*CommentGroup {
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

		// A blank line (line gap > 1) between comments starts a new group
		if curr.Start.Line-prev.End.Line > 1 {
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
