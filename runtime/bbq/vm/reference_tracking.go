package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/interpreter"
)

type ReferencedResourceKindedValues map[atree.ValueID]map[*EphemeralReferenceValue]struct{}

func maybeTrackReferencedResourceKindedValue(conf *Config, value Value) {
	referenceValue, ok := value.(*EphemeralReferenceValue)
	if !ok {
		return
	}

	referenceTRackedValue, ok := referenceValue.Value.(ReferenceTrackedResourceKindedValue)
	if !ok {
		return
	}

	trackReferencedResourceKindedValue(
		conf,
		referenceTRackedValue.ValueID(),
		referenceValue,
	)
}

func trackReferencedResourceKindedValue(
	conf *Config,
	id atree.ValueID,
	value *EphemeralReferenceValue,
) {
	values := conf.referencedResourceKindedValues[id]
	if values == nil {
		values = map[*EphemeralReferenceValue]struct{}{}
		conf.referencedResourceKindedValues[id] = values
	}
	values[value] = struct{}{}
}

func checkInvalidatedResourceOrResourceReference(value Value) {

	// Unwrap SomeValue, to access references wrapped inside optionals.
	someValue, isSomeValue := value.(*SomeValue)
	for isSomeValue && someValue.value != nil {
		value = someValue.value
		someValue, isSomeValue = value.(*SomeValue)
	}

	switch value := value.(type) {
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
	conf *Config,
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
			conf,
			func(_ string, fieldValue Value) (resume bool) {
				invalidateReferencedResources(conf, fieldValue)
				// continue iteration
				return true
			},
		)
		valueID = value.ValueID()

	case *DictionaryValue:
		value.IterateReadOnlyLoaded(
			conf,
			func(_, value Value) (resume bool) {
				invalidateReferencedResources(conf, value)
				return true
			},
		)
		valueID = value.ValueID()

	case *ArrayValue:
		value.IterateReadOnlyLoaded(
			conf,
			func(element Value) (resume bool) {
				invalidateReferencedResources(conf, element)
				return true
			},
		)
		valueID = value.ValueID()

	case *SomeValue:
		invalidateReferencedResources(conf, value.value)
		return

	default:
		// skip non-container typed values.
		return
	}

	values := conf.referencedResourceKindedValues[valueID]
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
	delete(conf.referencedResourceKindedValues, valueID)
}
