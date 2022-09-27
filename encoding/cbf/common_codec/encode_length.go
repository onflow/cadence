/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package common_codec

import (
	"encoding/binary"
	"fmt"
	"io"
)

// EncodeLength encodes a non-negative length as a uint32.
// It uses 4 bytes.
func EncodeLength(w io.Writer, length int) (err error) {
	if length < 0 { // TODO is this safety check useful?
		return CodecError(fmt.Sprintf("cannot encode length below zero: %d", length))
	}

	l := uint32(length)

	return binary.Write(w, binary.BigEndian, l)
}

func DecodeLength(r io.Reader) (length int, err error) {
	b := make([]byte, 4)

	bytesRead, err := r.Read(b)
	if err != nil {
		return
	}
	if bytesRead != 4 {
		err = CodecError("EOF when reading length")
		return
	}

	asUint32 := binary.BigEndian.Uint32(b)
	length = int(asUint32)
	return
}
