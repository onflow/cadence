package common_codec

import (
	"fmt"
	"io"
)

type LengthyWriter struct {
	w      io.Writer
	length int
}

func NewLengthyWriter(w io.Writer) LengthyWriter {
	return LengthyWriter{w: w}
}

func (l *LengthyWriter) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	l.length += n
	return
}

func (l *LengthyWriter) Len() int {
	return l.length
}

type LocatedReader struct {
	r        io.Reader
	location int
}

func NewLocatedReader(r io.Reader) LocatedReader {
	return LocatedReader{r: r}
}

func (l *LocatedReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	l.location += n
	return
}

func (l *LocatedReader) Location() int {
	return l.location
}

func Concat(deep ...[]byte) []byte {
	length := 0
	for _, b := range deep {
		length += len(b)
	}

	flat := make([]byte, 0, length)
	for _, b := range deep {
		flat = append(flat, b...)
	}

	return flat
}

type EncodedBool byte

const (
	EncodedBoolUnknown EncodedBool = iota
	EncodedBoolFalse
	EncodedBoolTrue
)

func EncodeBool(w io.Writer, boolean bool) (err error) {
	b := EncodedBoolFalse
	if boolean {
		b = EncodedBoolTrue
	}

	_, err = w.Write([]byte{byte(b)})
	return
}

func DecodeBool(r io.Reader) (boolean bool, err error) {
	b := make([]byte, 1)
	_, err = r.Read(b)
	if err != nil {
		return
	}

	switch EncodedBool(b[0]) {
	case EncodedBoolFalse:
		boolean = false
	case EncodedBoolTrue:
		boolean = true
	default:
		err = fmt.Errorf("invalid boolean value: %d", b[0])
	}

	return
}
