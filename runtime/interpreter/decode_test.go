package interpreter

import (
	"fmt"
	"math"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func TestThings(t *testing.T) {
	decMode, err := cbor.DecOptions{
		IntDec:           cbor.IntDecConvertNone,
		MaxArrayElements: math.MaxInt,
		MaxMapPairs:      math.MaxInt,
		MaxNestedLevels:  math.MaxInt16,
	}.DecMode()
	if err != nil {
		panic(err)
	}
	fmt.Println(decMode)
}
