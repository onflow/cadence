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

package runtime

import (
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

func newCheckHandler(i *Interface) sema.CheckHandlerFunc {
	return func(checker *sema.Checker, check func()) {
		reportMetric(
			check,
			*i,
			func(metrics Metrics, duration time.Duration) {
				metrics.ProgramChecked(checker.Location, duration)
			},
		)
	}
}

func newCapabilityBorrowHandler(handler stdlib.CapabilityControllerHandler) interpreter.CapabilityBorrowHandlerFunc {

	return func(
		context interpreter.BorrowCapabilityControllerContext,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		capabilityID interpreter.UInt64Value,
		wantedBorrowType *sema.ReferenceType,
		capabilityBorrowType *sema.ReferenceType,
	) interpreter.ReferenceValue {

		return stdlib.BorrowCapabilityController(
			context,
			locationRange,
			address,
			capabilityID,
			wantedBorrowType,
			capabilityBorrowType,
			handler,
		)
	}
}

func newCapabilityCheckHandler(handler stdlib.CapabilityControllerHandler) interpreter.CapabilityCheckHandlerFunc {
	return func(
		context interpreter.CheckCapabilityControllerContext,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		capabilityID interpreter.UInt64Value,
		wantedBorrowType *sema.ReferenceType,
		capabilityBorrowType *sema.ReferenceType,
	) interpreter.BoolValue {

		return stdlib.CheckCapabilityController(
			context,
			locationRange,
			address,
			capabilityID,
			wantedBorrowType,
			capabilityBorrowType,
			handler,
		)
	}
}

func newValidateAccountCapabilitiesGetHandler(i *Interface) interpreter.ValidateAccountCapabilitiesGetHandlerFunc {
	return func(
		context interpreter.AccountCapabilityGetValidationContext,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		path interpreter.PathValue,
		wantedBorrowType *sema.ReferenceType,
		capabilityBorrowType *sema.ReferenceType,
	) (bool, error) {

		return (*i).ValidateAccountCapabilitiesGet(
			context,
			locationRange,
			address,
			path,
			wantedBorrowType,
			capabilityBorrowType,
		)
	}
}

func newValidateAccountCapabilitiesPublishHandler(i *Interface) interpreter.ValidateAccountCapabilitiesPublishHandlerFunc {
	return func(
		context interpreter.AccountCapabilityPublishValidationContext,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		path interpreter.PathValue,
		capabilityBorrowType *interpreter.ReferenceStaticType,
	) (bool, error) {

		return (*i).ValidateAccountCapabilitiesPublish(
			context,
			locationRange,
			address,
			path,
			capabilityBorrowType,
		)
	}
}

func configureVersionedFeatures(i Interface) {
	minimumRequiredVersion, err := i.MinimumRequiredVersion()
	if err != nil {
		panic(err)
	}

	// No feature flags yet
	_ = minimumRequiredVersion
}

func newOnRecordTraceHandler(i *Interface) interpreter.OnRecordTraceFunc {
	return func(
		interpreter *interpreter.Interpreter,
		functionName string,
		duration time.Duration,
		attrs []attribute.KeyValue,
	) {
		(*i).RecordTrace(
			functionName,
			interpreter.Location,
			duration,
			attrs,
		)
	}
}

func newUUIDHandler(i *Interface) interpreter.UUIDHandlerFunc {
	return func() (uuid uint64, err error) {
		return (*i).GenerateUUID()
	}
}

func newInjectedCompositeFieldsHandler(accountHandler stdlib.AccountHandler) interpreter.InjectedCompositeFieldsHandlerFunc {
	return func(
		context interpreter.AccountCreationContext,
		location Location,
		_ string,
		compositeKind common.CompositeKind,
	) map[string]interpreter.Value {

		switch compositeKind {
		case common.CompositeKindContract:
			var address Address

			switch location := location.(type) {
			case common.AddressLocation:
				address = location.Address
			default:
				return nil
			}

			addressValue := interpreter.NewAddressValue(
				context,
				address,
			)

			return map[string]interpreter.Value{
				sema.ContractAccountFieldName: stdlib.NewAccountReferenceValue(
					context,
					accountHandler,
					addressValue,
					interpreter.FullyEntitledAccountAccess,
					interpreter.EmptyLocationRange,
				),
			}
		}

		return nil
	}
}

func newResourceOwnerChangedHandler(i *Interface) interpreter.OnResourceOwnerChangeFunc {
	return func(
		interpreter *interpreter.Interpreter,
		resource *interpreter.CompositeValue,
		oldOwner common.Address,
		newOwner common.Address,
	) {
		(*i).ResourceOwnerChanged(
			interpreter,
			resource,
			oldOwner,
			newOwner,
		)
	}
}

func newOnEventEmittedHandler(i *Interface) interpreter.OnEventEmittedFunc {
	return func(
		context interpreter.ValueExportContext,
		locationRange interpreter.LocationRange,
		eventType *sema.CompositeType,
		eventFields []interpreter.Value,
	) error {
		EmitEventFields(
			context,
			locationRange,
			eventType,
			eventFields,
			(*i).EmitEvent,
		)

		return nil
	}
}
