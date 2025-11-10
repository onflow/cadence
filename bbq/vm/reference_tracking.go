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

package vm

import (
	"sync"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/interpreter"
)

type ReferencedResourceKindedValues map[atree.ValueID]ReferenceSet

type ReferenceSet = map[*interpreter.EphemeralReferenceValue]struct{}

var referenceSetPool = sync.Pool{
	New: func() any {
		return ReferenceSet{}
	},
}

func newReferenceSet() ReferenceSet {
	set := referenceSetPool.Get().(ReferenceSet)
	return set
}

func releaseReferenceSet(set ReferenceSet) {
	if set == nil {
		return
	}
	clear(set)
	referenceSetPool.Put(set)
}

func (c *Context) MaybeTrackReferencedResourceKindedValue(referenceValue *interpreter.EphemeralReferenceValue) {
	referenceTrackedValue, ok := referenceValue.Value.(interpreter.ReferenceTrackedResourceKindedValue)
	if !ok {
		return
	}

	id := referenceTrackedValue.ValueID()

	values := c.referencedResourceKindedValues[id]
	if values == nil {
		values = newReferenceSet()
		if c.referencedResourceKindedValues == nil {
			c.referencedResourceKindedValues = make(ReferencedResourceKindedValues)
		}
		c.referencedResourceKindedValues[id] = values
	}
	values[referenceValue] = struct{}{}
}

func (c *Context) ClearReferencedResourceKindedValues(valueID atree.ValueID) {
	set := c.referencedResourceKindedValues[valueID]
	releaseReferenceSet(set)
	delete(c.referencedResourceKindedValues, valueID)
}

func (c *Context) ReferencedResourceKindedValues(valueID atree.ValueID) ReferenceSet {
	return c.referencedResourceKindedValues[valueID]
}
