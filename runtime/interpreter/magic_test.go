package interpreter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrependMagic(t *testing.T) {

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t,
			[]byte{0x0, 0xCA, 0xDE, 0x0, 0x1},
			PrependMagic([]byte{}),
		)
	})

	t.Run("1, 2, 3", func(t *testing.T) {
		assert.Equal(t,
			[]byte{0x0, 0xCA, 0xDE, 0x0, 0x1, 1, 2, 3},
			PrependMagic([]byte{1, 2, 3}),
		)
	})
}
