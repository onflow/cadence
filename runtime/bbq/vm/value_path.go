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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/interpreter"
)

type PathValue struct {
	Domain     common.PathDomain
	Identifier string
}

var _ Value = PathValue{}

func (PathValue) isValue() {}

func (v PathValue) StaticType(common.MemoryGauge) StaticType {
	switch v.Domain {
	case common.PathDomainStorage:
		return interpreter.PrimitiveStaticTypeStoragePath
	case common.PathDomainPublic:
		return interpreter.PrimitiveStaticTypePublicPath
	case common.PathDomainPrivate:
		return interpreter.PrimitiveStaticTypePrivatePath
	default:
		panic(errors.NewUnreachableError())
	}
}

func (v PathValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v PathValue) String() string {
	return format.Path(
		v.Domain.Identifier(),
		v.Identifier,
	)
}
