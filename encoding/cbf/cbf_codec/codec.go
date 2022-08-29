package cbf_codec

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding"
	"github.com/onflow/cadence/runtime/common"
)

type CadenceBinaryFormatCodec struct{}

func (v CadenceBinaryFormatCodec) Encode(value cadence.Value) ([]byte, error) {
	return EncodeValue(value)
}

func (v CadenceBinaryFormatCodec) MustEncode(value cadence.Value) []byte {
	return MustEncode(value)
}

func (v CadenceBinaryFormatCodec) Decode(gauge common.MemoryGauge, bytes []byte) (cadence.Value, error) {
	return DecodeValue(gauge, bytes)
}

func (v CadenceBinaryFormatCodec) MustDecode(gauge common.MemoryGauge, bytes []byte) cadence.Value {
	return MustDecode(gauge, bytes)
}

var _ encoding.Codec = CadenceBinaryFormatCodec{}
