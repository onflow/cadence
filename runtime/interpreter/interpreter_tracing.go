/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
)

const (
	// common
	tracingFunctionPrefix = "function."
	tracingImportPrefix   = "import."

	// type prefixes
	tracingArrayPrefix      = "array."
	tracingDictionaryPrefix = "dictionary."
	tracingCompositePrefix  = "composite."

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
)

func (interpreter *Interpreter) reportFunctionTrace(functionName string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingFunctionPrefix+functionName, duration, nil)
}

func (interpreter *Interpreter) reportImportTrace(importPath string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingImportPrefix+importPath, duration, nil)
}

func prepareArrayAndMapValueTraceAttrs(typeInfo string, count int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int("count", count),
		attribute.String("type", typeInfo),
	}
}

func (interpreter *Interpreter) reportArrayValueConstructTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingConstructPostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueDeepRemoveTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingDeepRemovePostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueDestroyTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingDestroyPostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingTransferPostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueConformsToStaticTypeTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingConformsToStaticTypePostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueConstructTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingConstructPostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueDeepRemoveTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingDeepRemovePostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueDestroyTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingDestroyPostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingTransferPostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueConformsToStaticTypeTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingConformsToStaticTypePostfix, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueGetMemberTrace(typeInfo string, count int, name string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingGetMemberPrefix+name, duration, prepareArrayAndMapValueTraceAttrs(typeInfo, count))
}

func prepareCompositeValueTraceAttrs(owner, typeID, kind string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("owner", owner),
		attribute.String("typeID", typeID),
		attribute.String("kind", kind),
	}
}

func (interpreter *Interpreter) reportCompositeValueConstructTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingConstructPostfix, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueDeepRemoveTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingDeepRemovePostfix, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueDestroyTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingDestroyPostfix, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueTransferTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingTransferPostfix, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueConformsToStaticTypeTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingConformsToStaticTypePostfix, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueGetMemberTrace(owner, typeID, kind, name string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingGetMemberPrefix+name, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueSetMemberTrace(owner, typeID, kind, name string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingSetMemberPrefix+name, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueRemoveMemberTrace(owner, typeID, kind, name string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingRemoveMemberPrefix+name, duration, prepareCompositeValueTraceAttrs(owner, typeID, kind))
}
