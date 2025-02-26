/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package compiler

import (
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/interpreter"
)

type TypeGen[T any] interface {
	CompileType(staticType bbq.StaticType) T
}

type DecodedTypeGen struct {
}

var _ TypeGen[bbq.StaticType] = &DecodedTypeGen{}

func (d DecodedTypeGen) CompileType(staticType bbq.StaticType) bbq.StaticType {
	return staticType
}

type EncodedTypeGen struct {
}

var _ TypeGen[[]byte] = &EncodedTypeGen{}

func (d EncodedTypeGen) CompileType(staticType bbq.StaticType) []byte {
	bytes, err := interpreter.StaticTypeToBytes(staticType)
	if err != nil {
		panic(err)
	}
	return bytes
}
