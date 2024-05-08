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

type CachedStaticType struct {
	StaticType interpreter.StaticType
	Error      error
}

type StaticTypeCache interface {
	Get(typeID interpreter.TypeID) (CachedStaticType, bool)
	Set(
		typeID interpreter.TypeID,
		staticType interpreter.StaticType,
		err error,
	)
}

type DefaultStaticTypeCache struct {
	entries sync.Map
}

func NewDefaultStaticTypeCache() *DefaultStaticTypeCache {
	return &DefaultStaticTypeCache{}
}

func (c *DefaultStaticTypeCache) Get(typeID interpreter.TypeID) (CachedStaticType, bool) {
	v, ok := c.entries.Load(typeID)
	if !ok {
		return CachedStaticType{}, false
	}
	return v.(CachedStaticType), true
}

func (c *DefaultStaticTypeCache) Set(
	typeID interpreter.TypeID,
	staticType interpreter.StaticType,
	err error,
) {
	c.entries.Store(typeID, CachedStaticType{
		StaticType: staticType,
		Error:      err,
	})
}
