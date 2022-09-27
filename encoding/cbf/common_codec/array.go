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
	"io"
)

func EncodeArray[T any](w io.Writer, arr []T, encodeFn func(T) error) (err error) {
	err = EncodeBool(w, arr == nil)
	if arr == nil || err != nil {
		return
	}

	err = EncodeLength(w, len(arr))
	if err != nil {
		return
	}

	for _, element := range arr {
		err = encodeFn(element)
		if err != nil {
			return
		}
	}

	return
}

func DecodeArray[T any](r io.Reader, maxSize int, decodeFn func() (T, error)) (arr []T, err error) {
	isNil, err := DecodeBool(r)
	if isNil || err != nil {
		return
	}

	length, err := DecodeLength(r, maxSize)
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
