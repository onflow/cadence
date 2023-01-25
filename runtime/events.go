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

package runtime

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func emitEventValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	event *interpreter.CompositeValue,
	emitEvent func(cadence.Event) error,
) {
	fields := make([]exportableValue, len(eventType.ConstructorParameters))

	for i, parameter := range eventType.ConstructorParameters {
		value := event.GetField(inter, locationRange, parameter.Identifier)
		fields[i] = newExportableValue(value, inter)
	}

	emitEventFields(inter, locationRange, eventType, fields, emitEvent)
}

func emitEventFields(
	gauge common.MemoryGauge,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	eventFields []exportableValue,
	emitEvent func(cadence.Event) error,
) {
	actualLen := len(eventFields)
	expectedLen := len(eventType.ConstructorParameters)

	if actualLen != expectedLen {
		panic(errors.NewDefaultUserError(
			"event emission value mismatch: event %s: expected %d, got %d",
			eventType.QualifiedString(),
			expectedLen,
			actualLen,
		))
	}

	eventValue := exportableEvent{
		Type:   eventType,
		Fields: eventFields,
	}

	exportedEvent, err := exportEvent(
		gauge,
		eventValue,
		locationRange,
		seenReferences{},
	)
	if err != nil {
		panic(err)
	}

	wrapPanic(func() {
		err = emitEvent(exportedEvent)
	})
	if err != nil {
		panic(err)
	}
}
