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

package format

import (
	"fmt"
)

func StorageCapability(borrowType string, id string, address string, path string) string {
	var typeArgument string
	if borrowType != "" {
		typeArgument = fmt.Sprintf("<%s>", borrowType)
	}

	return fmt.Sprintf(
		"Capability%s(id: %s, address: %s, path: %s)",
		typeArgument,
		id,
		address,
		path,
	)
}

func StorageCapabilityController(borrowType string, capabilityID string, target string) string {
	return fmt.Sprintf(
		"StorageCapabilityController(borrowType: %s, capabilityID: %s, target: %s)",
		borrowType,
		capabilityID,
		target,
	)
}

func AccountCapabilityController(borrowType string, capabilityID string) string {
	return fmt.Sprintf(
		"AccountCapabilityController(borrowType: %s, capabilityID: %s)",
		borrowType,
		capabilityID,
	)
}
