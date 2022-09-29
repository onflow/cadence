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

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
)

func EncodeAddress[Address common.Address | cadence.Address](w io.Writer, a Address) (err error) {
	_, err = w.Write(a[:])
	return
}

func DecodeAddress(r io.Reader) (a common.Address, err error) {
	bytes := make([]byte, common.AddressLength)

	bytesRead, err := r.Read(bytes)
	if err != nil {
		return
	}
	if bytesRead != common.AddressLength {
		err = CodecError("EOF when reading address")
		return
	}

	return common.BytesToAddress(bytes)
}
