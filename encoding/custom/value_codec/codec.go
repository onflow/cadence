package value_codec

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding"
	"github.com/onflow/cadence/runtime/common"
)

type ValueCodec struct{}

func (v ValueCodec) Encode(value cadence.Value) ([]byte, error) {
	return Encode(value)
}

func (v ValueCodec) MustEncode(value cadence.Value) []byte {
	return MustEncode(value)
}

func (v ValueCodec) Decode(gauge common.MemoryGauge, bytes []byte) (cadence.Value, error) {
	return Decode(gauge, bytes)
}

func (v ValueCodec) MustDecode(gauge common.MemoryGauge, bytes []byte) cadence.Value {
	return MustDecode(gauge, bytes)
}

var _ encoding.Codec = ValueCodec{}
