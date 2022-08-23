package json

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding"
	"github.com/onflow/cadence/runtime/common"
)

type JsonCodec struct{}

func (v JsonCodec) Encode(value cadence.Value) ([]byte, error) {
	return Encode(value)
}

func (v JsonCodec) MustEncode(value cadence.Value) []byte {
	return MustEncode(value)
}

func (v JsonCodec) Decode(gauge common.MemoryGauge, bytes []byte) (cadence.Value, error) {
	return Decode(gauge, bytes)
}

func (v JsonCodec) MustDecode(gauge common.MemoryGauge, bytes []byte) cadence.Value {
	return MustDecode(gauge, bytes)
}

var _ encoding.Codec = JsonCodec{}
