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

	"github.com/onflow/cadence/common"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// common
	tracingFunctionPrefix = "function."
	tracingImportPrefix   = "import."

	// type prefixes
	tracingArrayPrefix              = "array."
	tracingDictionaryPrefix         = "dictionary."
	tracingCompositePrefix          = "composite."
	tracingEphemeralReferencePrefix = "reference."

	// Value operation postfixes
	tracingConstructPostfix            = "construct"
	tracingTransferPostfix             = "transfer"
	tracingConformsToStaticTypePostfix = "conformsToStaticType"
	tracingDeepRemovePostfix           = "deepRemove"
	tracingDestroyPostfix              = "destroy"
	tracingCastPostfix                 = "cast"
	tracingBinaryOpPostfix             = "operation."

	// MemberAccessible operation prefixes
	tracingGetMemberPrefix    = "getMember."
	tracingSetMemberPrefix    = "setMember."
	tracingRemoveMemberPrefix = "removeMember."
)

// OnRecordTraceFunc is a function that records a trace.
type OnRecordTraceFunc func(
	executer Traceable,
	operationName string,
	duration time.Duration,
	attrs []attribute.KeyValue,
)

type Traceable interface {
	GetLocation() common.Location
}

var _ Traceable = &Interpreter{}

type Tracer struct {
	// OnRecordTrace is triggered when a trace is recorded
	OnRecordTrace OnRecordTraceFunc
	// TracingEnabled determines if tracing is enabled.
	// Tracing reports certain operations, e.g. composite value transfers
	TracingEnabled bool
}

func (tracer Tracer) reportFunctionTrace(executer Traceable, functionName string, duration time.Duration) {
	tracer.OnRecordTrace(executer, tracingFunctionPrefix+functionName, duration, nil)
}

func (tracer Tracer) reportImportTrace(executer Traceable, importPath string, duration time.Duration) {
	tracer.OnRecordTrace(executer, tracingImportPrefix+importPath, duration, nil)
}

func prepareArrayAndMapValueTraceAttrs(typeInfo string, count int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int("count", count),
		attribute.String("type", typeInfo),
	}
}

func (tracer Tracer) reportArrayValueConstructTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingArrayPrefix+tracingConstructPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportArrayValueDeepRemoveTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingArrayPrefix+tracingDeepRemovePostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportArrayValueDestroyTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingArrayPrefix+tracingDestroyPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportArrayValueTransferTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingArrayPrefix+tracingTransferPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportArrayValueConformsToStaticTypeTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingArrayPrefix+tracingConformsToStaticTypePostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportDictionaryValueConstructTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingDictionaryPrefix+tracingConstructPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportDictionaryValueDeepRemoveTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingDictionaryPrefix+tracingDeepRemovePostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportDictionaryValueDestroyTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingDictionaryPrefix+tracingDestroyPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportDictionaryValueTransferTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingDictionaryPrefix+tracingTransferPostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportDictionaryValueConformsToStaticTypeTrace(
	executer Traceable,
	typeInfo string,
	count int,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingDictionaryPrefix+tracingConformsToStaticTypePostfix,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func (tracer Tracer) reportDictionaryValueGetMemberTrace(
	executer Traceable,
	typeInfo string,
	count int,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingDictionaryPrefix+tracingGetMemberPrefix+name,
		duration,
		prepareArrayAndMapValueTraceAttrs(typeInfo, count),
	)
}

func prepareCompositeValueTraceAttrs(owner, typeID, kind string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("owner", owner),
		attribute.String("typeID", typeID),
		attribute.String("kind", kind),
	}
}

func (tracer Tracer) reportCompositeValueConstructTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingConstructPostfix,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportCompositeValueDeepRemoveTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingDeepRemovePostfix,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportCompositeValueDestroyTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingDestroyPostfix,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportCompositeValueTransferTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingTransferPostfix,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportCompositeValueConformsToStaticTypeTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingConformsToStaticTypePostfix,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportCompositeValueGetMemberTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingGetMemberPrefix+name,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportCompositeValueSetMemberTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingSetMemberPrefix+name,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportCompositeValueRemoveMemberTrace(
	executer Traceable,
	owner string,
	typeID string,
	kind string,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCompositePrefix+tracingRemoveMemberPrefix+name,
		duration,
		prepareCompositeValueTraceAttrs(owner, typeID, kind),
	)
}

func (tracer Tracer) reportTransferTrace(
	executer Traceable,
	targetType string,
	valueType string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingTransferPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("target type", targetType),
			attribute.String("value type", valueType),
		},
	)
}

func (tracer Tracer) reportCastingTrace(
	executer Traceable,
	targetType string,
	value string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingCastPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("target type", targetType),
			attribute.String("value", value),
		},
	)
}

func (tracer Tracer) reportEphemeralReferenceValueConstructTrace(
	executer Traceable,
	auth string,
	typeID string,
	value string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingEphemeralReferencePrefix+tracingConstructPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("auth", auth),
			attribute.String("typeID", typeID),
			attribute.String("value", value),
		},
	)
}

func (tracer Tracer) reportFunctionValueConstructTrace(
	executer Traceable,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingFunctionPrefix+tracingConstructPostfix,
		duration,
		nil,
	)
}

func (tracer Tracer) reportOpTrace(
	executer Traceable,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingBinaryOpPostfix+name,
		duration,
		nil,
	)
}
