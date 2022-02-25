/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

const (
	// common
	tracingFunctionPrefix = "function."
	tracingImportPrefix   = "import."

	// type prefixes
	tracingArrayPrefix      = "array."
	tracingDictionaryPrefix = "dictionary."
	tracingCompositePrefix  = "composite."

	// Value operation prefixes
	tracingConstructPrefix             = "construct."
	tracingTransferPrefix              = "transfer."
	tracingConformsToDynamicTypePrefix = "conformsToDynamicType."
	tracingClonePrefix                 = "clone."
	tracingDeepRemovePrefix            = "deepRemove."
	tracingDestroyPrefix               = "distroy."

	// ValueIndexable operation prefixes
	// TODO enable these
	// tracingGetKeyPrefix    = "getKey."
	// tracingSetKeyPrefix    = "setKey."
	// tracingRemoveKeyPrefix = "removeKey."
	// tracingInsertKeyPrefix = "insertKey."

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

func prepareArrayAndMapValueTraceLogs(typeInfo string, count int) []opentracing.LogRecord {
	return []opentracing.LogRecord{
		{
			Timestamp: time.Now(),
			Fields: []log.Field{
				log.Int("count", count),
				log.String("type", typeInfo),
			},
		},
	}
}

func (interpreter *Interpreter) reportArrayValueConstructTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingConstructPrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueCloneTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingClonePrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueDeepRemoveTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingDeepRemovePrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueDestroyTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingDestroyPrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingTransferPrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueConformsToDynamicTypeTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingConformsToDynamicTypePrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueConstructTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingConstructPrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueCloneTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingClonePrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueDeepRemoveTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingDeepRemovePrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueDestroyTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingDestroyPrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingTransferPrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueConformsToDynamicTypeTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingConformsToDynamicTypePrefix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func prepareCompositeValueTraceLogs(owner, typeID, kind string) []opentracing.LogRecord {
	return []opentracing.LogRecord{
		{
			Timestamp: time.Now(),
			Fields: []log.Field{
				log.String("owner", owner),
				log.String("typeID", typeID),
				log.String("kind", kind),
			},
		},
	}
}

func (interpreter *Interpreter) reportCompositeValueConstructTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingConstructPrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueCloneTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingClonePrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueDeepRemoveTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingDeepRemovePrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueDestroyTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingDestroyPrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueTransferTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingTransferPrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueConformsToDynamicTypeTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingConformsToDynamicTypePrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueGetMemberTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingGetMemberPrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueSetMemberTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingSetMemberPrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueRemoveMemberTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingCompositePrefix+tracingRemoveMemberPrefix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}
