package cbf_codec

import "github.com/onflow/cadence/encoding/cbf/common_codec"

func EncodeArray[T any](e *Encoder, arr []T, encodeFn func(T) error) (err error) {
	// TODO save a bit in the array length for nil check?
	err = common_codec.EncodeBool(&e.w, arr == nil)
	if arr == nil || err != nil {
		return
	}

	err = common_codec.EncodeLength(&e.w, len(arr))
	if err != nil {
		return
	}

	for _, element := range arr {
		// TODO does this need to include pointer logic for recursive types in arrays to be handled correctly?
		err = encodeFn(element)
		if err != nil {
			return
		}
	}

	return
}

func DecodeArray[T any](d *Decoder, decodeFn func() (T, error)) (arr []T, err error) {
	isNil, err := common_codec.DecodeBool(&d.r)
	if isNil || err != nil {
		return
	}

	length, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}

	arr = make([]T, length)
	for i := 0; i < length; i++ {
		var element T
		element, err = decodeFn()
		if err != nil {
			return
		}

		arr[i] = element
	}

	return
}
