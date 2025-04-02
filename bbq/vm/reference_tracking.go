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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/interpreter"
)

type ReferencedResourceKindedValues map[atree.ValueID]map[*interpreter.EphemeralReferenceValue]struct{}

func (c *Config) MaybeTrackReferencedResourceKindedValue(referenceValue *interpreter.EphemeralReferenceValue) {
	referenceTrackedValue, ok := referenceValue.Value.(interpreter.ReferenceTrackedResourceKindedValue)
	if !ok {
		return
	}

	id := referenceTrackedValue.ValueID()

	values := c.referencedResourceKindedValues[id]
	if values == nil {
		values = map[*interpreter.EphemeralReferenceValue]struct{}{}
		c.referencedResourceKindedValues[id] = values
	}
	values[referenceValue] = struct{}{}
}

func (c *Config) CheckInvalidatedResourceOrResourceReference(
	value Value,
	locationRange interpreter.LocationRange,
) {

	switch value := value.(type) {
	case *interpreter.SomeValue:
		c.CheckInvalidatedResourceOrResourceReference(value.InnerValue(), locationRange)

	// TODO:
	//case ResourceKindedValue:
	//	if value.isInvalidatedResource(interpreter) {
	//		panic(InvalidatedResourceError{
	//			LocationRange: LocationRange{
	//				Location:    interpreter.Location,
	//				HasPosition: hasPosition,
	//			},
	//		})
	//	}
	case *interpreter.EphemeralReferenceValue:
		if value.Value == nil {
			panic(interpreter.InvalidatedResourceReferenceError{
				//LocationRange: interpreter.LocationRange{
				//	Location:    interpreter.Location,
				//	HasPosition: hasPosition,
				//},
			})
		} else {
			// If the value is there, check whether the referenced value is an invalidated one.
			// This step is not really needed, since reference tracking is supposed to clear the
			// `value.Value` if the referenced-value was moved/deleted.
			// However, have this as a second layer of defensive.
			c.CheckInvalidatedResourceOrResourceReference(value.Value, locationRange)
		}
	}
}

func (c *Config) ClearReferencedResourceKindedValues(valueID atree.ValueID) {
	delete(c.referencedResourceKindedValues, valueID)
}

func (c *Config) ReferencedResourceKindedValues(valueID atree.ValueID) map[*interpreter.EphemeralReferenceValue]struct{} {
	return c.referencedResourceKindedValues[valueID]
}
