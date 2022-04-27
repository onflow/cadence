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

package server

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/sema"
)

type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func encodeParameters(parameters []*sema.Parameter) []Parameter {

	encodedParameters := make([]Parameter, len(parameters))

	for i, parameter := range parameters {
		parameterType := runtime.ExportType(
			parameter.TypeAnnotation.Type,
			map[sema.TypeID]cadence.Type{},
		)

		var typeID string
		if parameterType != nil {
			typeID = parameterType.ID()
		}

		encodedParameters[i] = Parameter{
			Name: parameter.EffectiveArgumentLabel(),
			Type: typeID,
		}
	}

	return encodedParameters
}
