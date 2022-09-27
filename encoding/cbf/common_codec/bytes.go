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

import "io"

func EncodeBytes(w io.Writer, bytes []byte) (err error) {
	err = EncodeLength(w, len(bytes))
	if err != nil {
		return
	}
	_, err = w.Write(bytes)
	return
}

func DecodeBytes(r io.Reader) (bytes []byte, err error) {
	length, err := DecodeLength(r)
	if err != nil {
		return
	}

	bytes = make([]byte, length)

	bytesRead, err := r.Read(bytes)
	if err != nil {
		return
	}
	if bytesRead != length {
		err = CodecError("EOF when reading bytes")
		return
	}

	return
}
