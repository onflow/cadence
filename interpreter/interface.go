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

package interpreter

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type TypeConverter interface {
	MustConvertStaticToSemaType(staticType StaticType) sema.Type
}

var _ TypeConverter = &Interpreter{}

type SubTypeChecker interface {
	IsSubTypeOfSemaType(staticSubType StaticType, superType sema.Type) bool
}

var _ SubTypeChecker = &Interpreter{}

type StorageReader interface {
	ReadStored(
		storageAddress common.Address,
		domain common.StorageDomain,
		identifier StorageMapKey,
	) Value
}

var _ StorageReader = &Interpreter{}

type ValueStaticTypeContext interface {
	common.MemoryGauge
	StorageReader
	TypeConverter
	SubTypeChecker
}

var _ ValueStaticTypeContext = &Interpreter{}
