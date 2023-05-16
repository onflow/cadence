/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package interpreter

import (
	"encoding/binary"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
)

type Uint64AtreeValue uint64

var _ atree.Value = Uint64AtreeValue(0)
var _ atree.Storable = Uint64AtreeValue(0)

func (v Uint64AtreeValue) Storable(
	_ atree.SlabStorage,
	_ atree.Address,
	_ uint64,
) (
	atree.Storable,
	error,
) {
	return v, nil
}

func NewUint64AtreeValue(gauge common.MemoryGauge, s uint64) Uint64AtreeValue {
	common.UseMemory(gauge, UInt64MemoryUsage)
	return Uint64AtreeValue(s)
}

func (v Uint64AtreeValue) ByteSize() uint32 {
	return getUintCBORSize(uint64(v))
}

func (v Uint64AtreeValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Uint64AtreeValue) ChildStorables() []atree.Storable {
	return nil
}

func Uint64AtreeValueHashInput(v atree.Value, scratch []byte) ([]byte, error) {
	binary.BigEndian.PutUint64(scratch[:], uint64(v.(Uint64AtreeValue)))
	return scratch[:8], nil
}

func Uint64AtreeValueComparator(_ atree.SlabStorage, value atree.Value, otherStorable atree.Storable) (bool, error) {
	result := value.(Uint64AtreeValue) == otherStorable.(Uint64AtreeValue)
	return result, nil
}
