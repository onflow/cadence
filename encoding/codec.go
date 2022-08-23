package encoding

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
)

type Codec interface {
	Encode(cadence.Value) ([]byte, error)
	MustEncode(cadence.Value) []byte

	Decode(common.MemoryGauge, []byte) (cadence.Value, error)
	MustDecode(common.MemoryGauge, []byte) cadence.Value
}
