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

func EncodeString(w io.Writer, s string) (err error) {
	return EncodeBytes(w, []byte(s))
}

func DecodeString(r io.Reader, maxSize int) (s string, err error) {
	length, err := DecodeStringHeader(r, maxSize)
	if err != nil {
		return
	}

	return DecodeStringElements(r, length)
}

var DecodeStringHeader = DecodeBytesHeader

func DecodeStringElements(r io.Reader, length int) (s string, err error) {
	b, err := DecodeBytesElements(r, length)
	return string(b), err // string(nil) casts to empty string
}
