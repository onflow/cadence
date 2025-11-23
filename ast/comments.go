package ast

import (
	"bytes"
	"strings"

	"github.com/onflow/cadence/common"
)

type Comments struct {
	Leading  []*Comment `json:"-"`
	Trailing []*Comment `json:"-"`
}

var EmptyComments = Comments{}

// All combines Leading and Trailing comments in a single array.
func (c Comments) All() []*Comment {
	var comments []*Comment
	comments = append(comments, c.Leading...)
	comments = append(comments, c.Trailing...)
	return comments
}

// LeadingDocString prints the leading doc comments to string
func (c Comments) LeadingDocString() string {
	var s strings.Builder
	for _, comment := range c.Leading {
		if comment.IsDoc() {
			if s.Len() > 0 {
				s.WriteRune('\n')
			}
			s.Write(comment.Text())
		}
	}
	return s.String()
}

type Comment struct {
	source []byte
}

func NewComment(memoryGauge common.MemoryGauge, source []byte) *Comment {
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(source)))
	return &Comment{
		source: source,
	}
}

var blockCommentDocStringPrefix = []byte("/**")
var blockCommentStringPrefix = []byte("/*")
var lineCommentDocStringPrefix = []byte("///")
var lineCommentStringPrefix = []byte("//")
var blockCommentStringSuffix = []byte("*/")

func (c Comment) Multiline() bool {
	return bytes.HasPrefix(c.source, blockCommentStringPrefix)
}

func (c Comment) IsDoc() bool {
	if c.Multiline() {
		return bytes.HasPrefix(c.source, blockCommentDocStringPrefix)
	} else {
		return bytes.HasPrefix(c.source, lineCommentDocStringPrefix)
	}
}

var commentPrefixes = [][]byte{
	blockCommentDocStringPrefix, // must be before blockCommentStringPrefix
	blockCommentStringPrefix,
	lineCommentDocStringPrefix, // must be before lineCommentStringPrefix
	lineCommentStringPrefix,
}

var commentSuffixes = [][]byte{
	blockCommentStringSuffix,
}

// Text without opening/closing comment characters /*, /**, */, //
func (c Comment) Text() []byte {
	withoutPrefixes := cutOptionalPrefixes(c.source, commentPrefixes)
	return cutOptionalSuffixes(withoutPrefixes, commentSuffixes)
}

func cutOptionalPrefixes(input []byte, prefixes [][]byte) (output []byte) {
	output = input
	for _, prefix := range prefixes {
		cut, _ := bytes.CutPrefix(output, prefix)
		output = cut
	}
	return
}

func cutOptionalSuffixes(input []byte, suffixes [][]byte) (output []byte) {
	output = input
	for _, suffix := range suffixes {
		cut, _ := bytes.CutSuffix(output, suffix)
		output = cut
	}
	return
}
