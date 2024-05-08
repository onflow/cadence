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

package migrations

import (
	"sync"

	"github.com/onflow/cadence/runtime/interpreter"
)

type StaticTypeCache interface {
	Get(typeID interpreter.TypeID) (interpreter.StaticType, bool)
	Set(typeID interpreter.TypeID, ty interpreter.StaticType)
}

type DefaultStaticTypeCache struct {
	entries sync.Map
}

func NewDefaultStaticTypeCache() *DefaultStaticTypeCache {
	return &DefaultStaticTypeCache{}
}

func (c *DefaultStaticTypeCache) Get(typeID interpreter.TypeID) (interpreter.StaticType, bool) {
	v, ok := c.entries.Load(typeID)
	if !ok {
		return nil, false
	}
	staticType, _ := v.(interpreter.StaticType)
	return staticType, true
}

func (c *DefaultStaticTypeCache) Set(typeID interpreter.TypeID, ty interpreter.StaticType) {
	c.entries.Store(typeID, ty)
}
