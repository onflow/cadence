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
	tracingVariablePrefix = "variable."

	// type prefixes
	tracingArrayPrefix              = "array."
	tracingDictionaryPrefix         = "dictionary."
	tracingCompositePrefix          = "composite."
	tracingEphemeralReferencePrefix = "reference."

	// Value operation postfixes
	tracingConstructPostfix            = "construct."
	tracingTransferPostfix             = "transfer"
	tracingConformsToStaticTypePostfix = "conformsToStaticType"
	tracingDeepRemovePostfix           = "deepRemove"
	tracingDestroyPostfix              = "destroy"
	tracingCastPostfix                 = "cast"
	tracingOpPostfix                   = "operation."
	tracingReadPostfix                 = "read."
	tracingWritePostfix                = "write."

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

func (tracer Tracer) ReportFunctionTrace(executer Traceable, functionName string, duration time.Duration) {
	tracer.OnRecordTrace(executer, tracingFunctionPrefix+functionName, duration, nil)
}

func (tracer Tracer) ReportImportTrace(executer Traceable, importPath string, duration time.Duration) {
	tracer.OnRecordTrace(executer, tracingImportPrefix+importPath, duration, nil)
}

func prepareArrayAndMapValueTraceAttrs(typeInfo string, count int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int("count", count),
		attribute.String("type", typeInfo),
	}
}

func (tracer Tracer) ReportArrayValueConstructTrace(
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

func (tracer Tracer) ReportArrayValueDeepRemoveTrace(
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

func (tracer Tracer) ReportArrayValueDestroyTrace(
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

func (tracer Tracer) ReportArrayValueTransferTrace(
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

func (tracer Tracer) ReportArrayValueConformsToStaticTypeTrace(
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

func (tracer Tracer) ReportDictionaryValueConstructTrace(
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

func (tracer Tracer) ReportDictionaryValueDeepRemoveTrace(
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

func (tracer Tracer) ReportDictionaryValueDestroyTrace(
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

func (tracer Tracer) ReportDictionaryValueTransferTrace(
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

func (tracer Tracer) ReportDictionaryValueConformsToStaticTypeTrace(
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

func (tracer Tracer) ReportDictionaryValueGetMemberTrace(
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

func (tracer Tracer) ReportCompositeValueConstructTrace(
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

func (tracer Tracer) ReportCompositeValueDeepRemoveTrace(
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

func (tracer Tracer) ReportCompositeValueDestroyTrace(
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

func (tracer Tracer) ReportCompositeValueTransferTrace(
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

func (tracer Tracer) ReportCompositeValueConformsToStaticTypeTrace(
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

func (tracer Tracer) ReportCompositeValueGetMemberTrace(
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

func (tracer Tracer) ReportCompositeValueSetMemberTrace(
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

func (tracer Tracer) ReportCompositeValueRemoveMemberTrace(
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

func (tracer Tracer) ReportTransferTrace(
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

func (tracer Tracer) ReportCastingTrace(
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

func (tracer Tracer) ReportEphemeralReferenceValueConstructTrace(
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

func (tracer Tracer) ReportFunctionValueConstructTrace(
	executer Traceable,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingFunctionPrefix+tracingConstructPostfix+name,
		duration,
		nil,
	)
}

func (tracer Tracer) ReportOpTrace(
	executer Traceable,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingOpPostfix+name,
		duration,
		nil,
	)
}

func (tracer Tracer) ReportVariableReadTrace(
	executer Traceable,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingVariablePrefix+tracingReadPostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("name", name),
		},
	)
}

func (tracer Tracer) ReportVariableWriteTrace(
	executer Traceable,
	name string,
	duration time.Duration,
) {
	tracer.OnRecordTrace(executer,
		tracingVariablePrefix+tracingWritePostfix,
		duration,
		[]attribute.KeyValue{
			attribute.String("name", name),
		},
	)
}
