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
	"math/big"

	"github.com/onflow/cadence/runtime/common"
)

func EncodeBigInt(w io.Writer, i *big.Int) (err error) {
	isNegative := i.Sign() == -1
	err = EncodeBool(w, isNegative)
	if err != nil {
		return
	}

	return EncodeBytes(w, i.Bytes())
}

func DecodeBigInt(r io.Reader, maxSize int, memoryGauge common.MemoryGauge) (i *big.Int, err error) {
	isNegative, err := DecodeBool(r)
	if err != nil {
		return
	}

	length, err := DecodeBytesHeader(r, maxSize)
	if err != nil {
		return
	}

	b, err := DecodeBytesElements(r, length)
	if err != nil {
		return
	}

	common.UseMemory(memoryGauge, common.NewCadenceBigIntMemoryUsage(length))

	i = big.NewInt(0)
	i.SetBytes(b)
	if isNegative {
		i.Neg(i)
	}
	return
}
