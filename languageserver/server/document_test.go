package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocument_Offset(t *testing.T) {

	doc := Document{Text: "abcd\nefghijk\nlmno\npqr"}

	assert.Equal(t, 1, doc.Offset(1, 1))
	assert.Equal(t, 7, doc.Offset(2, 2))
	assert.Equal(t, 19, doc.Offset(4, 1))
}

func TestDocument_HasAnyPrecedingStringsAtPosition(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		doc := Document{Text: "  pub \t  \n  f"}

		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"pub"}, 2, 1))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"pub"}, 2, 2))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"pub"}, 2, 3))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"access(self)", "pub"}, 2, 2))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"access(self)", "pub"}, 1, 6))
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		doc := Document{Text: "  pub \t  \n  f"}

		assert.False(t, doc.HasAnyPrecedingStringsAtPosition([]string{"access(self)"}, 2, 2))
	})
}
