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

package main

import (
	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func prepareType(value interpreter.Value, inter *interpreter.Interpreter) (result any, description string) {
	staticType := value.StaticType(inter)

	defer func() {
		if recover() != nil {
			typeID := staticType.ID()
			result = typeID
			description = string(typeID)
		}
	}()

	semaType, err := inter.ConvertStaticToSemaType(staticType)
	if err != nil {
		typeID := staticType.ID()
		return typeID, string(typeID)
	}

	cadenceType := runtime.ExportType(
		semaType,
		map[sema.TypeID]cadence.Type{},
	)

	return jsoncdc.PrepareType(
		cadenceType,
		jsoncdc.TypePreparationResults{},
	), semaType.QualifiedString()
}
