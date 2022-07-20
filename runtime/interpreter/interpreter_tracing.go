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
	interpreter.Config.OnRecordTrace(interpreter, tracingFunctionPrefix+functionName, duration, nil)
}

func (interpreter *Interpreter) reportImportTrace(importPath string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingImportPrefix+importPath, duration, nil)
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
	interpreter.Config.OnRecordTrace(interpreter, tracingArrayPrefix+tracingConstructPostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueDeepRemoveTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingArrayPrefix+tracingDeepRemovePostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueDestroyTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingArrayPrefix+tracingDestroyPostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingArrayPrefix+tracingTransferPostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportArrayValueConformsToStaticTypeTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingArrayPrefix+tracingConformsToStaticTypePostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueConstructTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingDictionaryPrefix+tracingConstructPostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueDeepRemoveTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingDictionaryPrefix+tracingDeepRemovePostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueDestroyTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingDictionaryPrefix+tracingDestroyPostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingDictionaryPrefix+tracingTransferPostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueConformsToStaticTypeTrace(typeInfo string, count int, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingDictionaryPrefix+tracingConformsToStaticTypePostfix, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
}

func (interpreter *Interpreter) reportDictionaryValueGetMemberTrace(typeInfo string, count int, name string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingDictionaryPrefix+tracingGetMemberPrefix+name, duration, prepareArrayAndMapValueTraceLogs(typeInfo, count))
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
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingConstructPostfix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueDeepRemoveTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingDeepRemovePostfix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueDestroyTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingDestroyPostfix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueTransferTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingTransferPostfix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueConformsToStaticTypeTrace(owner, typeID, kind string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingConformsToStaticTypePostfix, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueGetMemberTrace(owner, typeID, kind, name string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingGetMemberPrefix+name, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueSetMemberTrace(owner, typeID, kind, name string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingSetMemberPrefix+name, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}

func (interpreter *Interpreter) reportCompositeValueRemoveMemberTrace(owner, typeID, kind, name string, duration time.Duration) {
	interpreter.Config.OnRecordTrace(interpreter, tracingCompositePrefix+tracingRemoveMemberPrefix+name, duration, prepareCompositeValueTraceLogs(owner, typeID, kind))
}
