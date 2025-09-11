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
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/errors"
)

const (
	// common
	tracingFunctionPrefix = "function."
	tracingImportPrefix   = "import."

	// type prefixes
	tracingArrayPrefix      = "array."
	tracingDictionaryPrefix = "dictionary."
	tracingCompositePrefix  = "composite."
	tracingAtreeMapPrefix   = "atreeMap."

	// Value operation postfixes
	tracingConstructPostfix            = "construct"
	tracingTransferPostfix             = "transfer"
	tracingConformsToStaticTypePostfix = "conformsToStaticType"
	tracingDeepRemovePostfix           = "deepRemove"
	tracingDestroyPostfix              = "destroy"

	// MemberAccessible operation prefixes
	tracingGetMemberPrefix    = "getMember."
	tracingSetMemberPrefix    = "setMember."
	tracingRemoveMemberPrefix = "removeMember."

	tracingAtreeMapNewFromBatchDataPostfix = "newFromBatchData"
)

type Tracer interface {
	ReportFunctionTrace(functionName string, duration time.Duration)
	ReportImportTrace(importPath string, duration time.Duration)

	ReportArrayValueConstructTrace(valueID string, typeID string, duration time.Duration)
	ReportArrayValueTransferTrace(valueID string, typeID string, duration time.Duration)
	ReportArrayValueDeepRemoveTrace(valueID string, typeID string, duration time.Duration)
	ReportArrayValueDestroyTrace(valueID string, typeID string, duration time.Duration)
	ReportArrayValueConformsToStaticTypeTrace(valueID string, typeID string, duration time.Duration)

	ReportDictionaryValueConstructTrace(valueID string, typeID string, duration time.Duration)
	ReportDictionaryValueTransferTrace(valueID string, typeID string, duration time.Duration)
	ReportDictionaryValueDeepRemoveTrace(valueID string, typeID string, duration time.Duration)
	ReportDictionaryValueDestroyTrace(valueID string, typeID string, duration time.Duration)
	ReportDictionaryValueConformsToStaticTypeTrace(valueID string, typeID string, duration time.Duration)

	ReportCompositeValueConstructTrace(valueID string, typeID string, kind string, duration time.Duration)
	ReportCompositeValueTransferTrace(valueID string, typeID string, kind string, duration time.Duration)
	ReportCompositeValueDeepRemoveTrace(valueID string, typeID string, kind string, duration time.Duration)
	ReportCompositeValueDestroyTrace(valueID string, typeID string, kind string, duration time.Duration)
	ReportCompositeValueConformsToStaticTypeTrace(valueID string, typeID string, kind string, duration time.Duration)
	ReportCompositeValueGetMemberTrace(valueID string, typeID string, kind string, name string, duration time.Duration)
	ReportCompositeValueSetMemberTrace(valueID string, typeID string, kind string, name string, duration time.Duration)
	ReportCompositeValueRemoveMemberTrace(valueID string, typeID string, kind string, name string, duration time.Duration)

	ReportAtreeNewMapFromBatchData(valueID string, typeID string, seed uint64, duration time.Duration)
}

type CallbackTracer OnRecordTraceFunc

var _ Tracer = CallbackTracer(nil)

func (t CallbackTracer) ReportFunctionTrace(functionName string, duration time.Duration) {
	t(
		tracingFunctionPrefix+functionName,
		duration,
		nil,
	)
}

func (t CallbackTracer) ReportImportTrace(importPath string, duration time.Duration) {
	t(
		tracingImportPrefix+importPath,
		duration,
		nil,
	)
}

func prepareContainerValueTraceAttrs(valueID string, typeID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("value", valueID),
		attribute.String("type", typeID),
	}
}

func (t CallbackTracer) reportContainerTrace(
	traceName string,
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t(
		traceName,
		duration,
		prepareContainerValueTraceAttrs(valueID, typeID),
	)
}

func (t CallbackTracer) ReportArrayValueConstructTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingArrayPrefix+tracingConstructPostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportArrayValueDeepRemoveTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingArrayPrefix+tracingDeepRemovePostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportArrayValueDestroyTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingArrayPrefix+tracingDestroyPostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportArrayValueTransferTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingArrayPrefix+tracingTransferPostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportArrayValueConformsToStaticTypeTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingArrayPrefix+tracingConformsToStaticTypePostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportDictionaryValueConstructTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingDictionaryPrefix+tracingConstructPostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportDictionaryValueDeepRemoveTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingDictionaryPrefix+tracingDeepRemovePostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportDictionaryValueDestroyTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingDictionaryPrefix+tracingDestroyPostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportDictionaryValueTransferTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingDictionaryPrefix+tracingTransferPostfix,
		valueID,
		typeID,
		duration,
	)
}

func (t CallbackTracer) ReportDictionaryValueConformsToStaticTypeTrace(
	valueID string,
	typeID string,
	duration time.Duration,
) {
	t.reportContainerTrace(
		tracingDictionaryPrefix+tracingConformsToStaticTypePostfix,
		valueID,
		typeID,
		duration,
	)
}

func prepareCompositeValueTraceAttrs(valueID string, typeID string, kind string) []attribute.KeyValue {
	return append(
		prepareContainerValueTraceAttrs(valueID, typeID),
		attribute.String("kind", kind),
	)
}

func (t CallbackTracer) reportCompositeTrace(
	traceName string,
	valueID string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	t(
		traceName,
		duration,
		prepareCompositeValueTraceAttrs(valueID, typeID, kind),
	)
}

func (t CallbackTracer) ReportCompositeValueConstructTrace(
	valueID string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingConstructPostfix,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func (t CallbackTracer) ReportCompositeValueDeepRemoveTrace(
	valueID string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingDeepRemovePostfix,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func (t CallbackTracer) ReportCompositeValueDestroyTrace(
	valueID string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingDestroyPostfix,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func (t CallbackTracer) ReportCompositeValueTransferTrace(
	valueID string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingTransferPostfix,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func (t CallbackTracer) ReportCompositeValueConformsToStaticTypeTrace(
	valueID string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingConformsToStaticTypePostfix,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func (t CallbackTracer) ReportCompositeValueGetMemberTrace(
	valueID string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingGetMemberPrefix+name,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func (t CallbackTracer) ReportCompositeValueSetMemberTrace(
	valueID string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingSetMemberPrefix+name,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func (t CallbackTracer) ReportCompositeValueRemoveMemberTrace(
	valueID string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	t.reportCompositeTrace(
		tracingCompositePrefix+tracingRemoveMemberPrefix+name,
		valueID,
		typeID,
		kind,
		duration,
	)
}

func prepareAtreeMapTraceAttrs(valueID string, typeID string, seed uint64) []attribute.KeyValue {
	return append(
		prepareContainerValueTraceAttrs(valueID, typeID),
		// OpenTelemetry does not support unsigned integers, so we use Int64.
		// The conversion might overflow if the seed is too large,
		// but this information is only used for debugging purposes.
		attribute.Int64("seed", int64(seed)),
	)
}

func (t CallbackTracer) ReportAtreeNewMapFromBatchData(
	valueID string,
	typeID string,
	seed uint64,
	duration time.Duration,
) {
	t(
		tracingAtreeMapPrefix+tracingAtreeMapNewFromBatchDataPostfix,
		duration,
		prepareAtreeMapTraceAttrs(valueID, typeID, seed),
	)
}

type NoOpTracer struct{}

var _ Tracer = NoOpTracer{}

func (NoOpTracer) ReportFunctionTrace(_ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportImportTrace(_ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportArrayValueDeepRemoveTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportArrayValueTransferTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportArrayValueDestroyTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportArrayValueConstructTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportDictionaryValueTransferTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportArrayValueConformsToStaticTypeTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportDictionaryValueDestroyTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportDictionaryValueDeepRemoveTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueDeepRemoveTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportDictionaryValueGetMemberTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportDictionaryValueConstructTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportDictionaryValueConformsToStaticTypeTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueTransferTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueSetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueDestroyTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueGetMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueConstructTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueConformsToStaticTypeTrace(_ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportCompositeValueRemoveMemberTrace(_ string, _ string, _ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportDomainStorageMapDeepRemoveTrace(_ string, _ string, _ time.Duration) {
	panic(errors.NewUnreachableError())
}

func (NoOpTracer) ReportAtreeNewMapFromBatchData(_ string, _ string, _ uint64, _ time.Duration) {
	panic(errors.NewUnreachableError())
}
