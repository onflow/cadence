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

type ReferencedResourceKindedValues map[atree.ValueID]map[*EphemeralReferenceValue]struct{}

func maybeTrackReferencedResourceKindedValue(referenceTracker ReferenceTracker, value Value) {
	referenceValue, ok := value.(*EphemeralReferenceValue)
	if !ok {
		return
	}

	referenceTRackedValue, ok := referenceValue.Value.(ReferenceTrackedResourceKindedValue)
	if !ok {
		return
	}

	referenceTracker.TrackReferencedResourceKindedValue(
		referenceTRackedValue.ValueID(),
		referenceValue,
	)
}

func checkInvalidatedResourceOrResourceReference(value Value) {

	switch value := value.(type) {
	case *SomeValue:
		checkInvalidatedResourceOrResourceReference(value.value)

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
	case *EphemeralReferenceValue:
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
			checkInvalidatedResourceOrResourceReference(value.Value)
		}
	}
}

func invalidateReferencedResources(
	context TransferContext,
	value Value,
) {
	// skip non-resource typed values
	resourceKinded, ok := value.(ResourceKindedValue)
	if !ok || !resourceKinded.IsResourceKinded() {
		return
	}

	var valueID atree.ValueID

	switch value := resourceKinded.(type) {
	case *CompositeValue:
		value.ForEachReadOnlyLoadedField(
			context,
			func(_ string, fieldValue Value) (resume bool) {
				invalidateReferencedResources(context, fieldValue)
				// continue iteration
				return true
			},
		)
		valueID = value.ValueID()

	case *DictionaryValue:
		value.IterateReadOnlyLoaded(
			context,
			func(_, value Value) (resume bool) {
				invalidateReferencedResources(context, value)
				return true
			},
		)
		valueID = value.ValueID()

	case *ArrayValue:
		value.IterateReadOnlyLoaded(
			context,
			func(element Value) (resume bool) {
				invalidateReferencedResources(context, element)
				return true
			},
		)
		valueID = value.ValueID()

	case *SomeValue:
		invalidateReferencedResources(context, value.value)
		return

	default:
		// skip non-container typed values.
		return
	}

	values := context.ReferencedResourceKindedValues(valueID)
	if values == nil {
		return
	}

	for value := range values { //nolint:maprange
		value.Value = nil
	}

	// The old resource instances are already cleared/invalidated above.
	// So no need to track those stale resources anymore. We will not need to update/clear them again.
	// Therefore, remove them from the mapping.
	// This is only to allow GC. No impact to the behavior.
	context.ClearReferenceTracking(valueID)
}
