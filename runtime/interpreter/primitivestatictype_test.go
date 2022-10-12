package interpreter

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrimitiveStaticTypeSemaTypeConversion(t *testing.T) {

	t.Parallel()

	placeholderTypePattern := regexp.MustCompile("PrimitiveStaticType\\(\\d+\\)")

	test := func(ty PrimitiveStaticType) {
		t.Run(ty.String(), func(t *testing.T) {
			t.Parallel()

			semaType := ty.SemaType()
			ty2 := ConvertSemaToPrimitiveStaticType(nil, semaType)
			require.True(t, ty2.Equal(ty))
		})
	}

	for ty := PrimitiveStaticType(1); ty < PrimitiveStaticType_Count; ty++ {
		if placeholderTypePattern.MatchString(ty.String()) {
			continue
		}
		test(ty)
	}

}
