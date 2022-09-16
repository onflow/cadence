package common_codec

type CodecError string

var _ error = CodecError("")

func (c CodecError) Error() string {
	return string(c)
}
