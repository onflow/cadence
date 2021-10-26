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
	tracingFunctionPrefix   = "function."
	tracingImportPrefix     = "import."
	tracingArrayPrefix      = "array."
	tracingDictionaryPrefix = "dictionary."
	tracingTransferPrefix   = "transfer."
)

func (interpreter *Interpreter) reportFunctionTrace(functionName string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingFunctionPrefix+functionName, duration, nil)
}

func (interpreter *Interpreter) reportImportTrace(importPath string, duration time.Duration) {
	interpreter.onRecordTrace(interpreter, tracingImportPrefix+importPath, duration, nil)
}

func (interpreter *Interpreter) reportArrayValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	logs := []opentracing.LogRecord{
		{
			Timestamp: time.Now(),
			Fields: []log.Field{
				log.Int("count", count),
				log.String("type", typeInfo),
			},
		},
	}
	interpreter.onRecordTrace(interpreter, tracingArrayPrefix+tracingTransferPrefix, duration, logs)
}

func (interpreter *Interpreter) reportDictionaryValueTransferTrace(typeInfo string, count int, duration time.Duration) {
	logs := []opentracing.LogRecord{
		{
			Timestamp: time.Now(),
			Fields: []log.Field{
				log.Int("count", count),
				log.String("type", typeInfo),
			},
		},
	}
	interpreter.onRecordTrace(interpreter, tracingDictionaryPrefix+tracingTransferPrefix, duration, logs)
}
