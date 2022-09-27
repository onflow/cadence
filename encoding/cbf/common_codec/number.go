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
	"io"
)

func EncodeNumber[T int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64](w io.Writer, i T) (err error) {
	return binary.Write(w, binary.BigEndian, i)
}

func DecodeNumber[T int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64](r io.Reader) (i T, err error) {
	err = binary.Read(r, binary.BigEndian, &i)
	return
}
