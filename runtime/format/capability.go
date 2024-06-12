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

package format

import (
	"fmt"
)

// PathCapability returns the string representation of a path capability. Deprecated and removed in v1.0.0.
func DeprecatedPathCapability(borrowType string, address string, path string) string {
	var typeArgument string
	if borrowType != "" {
		typeArgument = fmt.Sprintf("<%s>", borrowType)
	}

	return fmt.Sprintf(
		"Capability%s(address: %s, path: %s)",
		typeArgument,
		address,
		path,
	)
}

func Capability(borrowType string, address string, id string) string {
	return fmt.Sprintf(
		"Capability<%s>(address: %s, id: %s)",
		borrowType,
		address,
		id,
	)
}

func StorageCapabilityController(borrowType string, capabilityID string, target string) string {
	return fmt.Sprintf(
		"StorageCapabilityController(borrowType: Type<%s>(), capabilityID: %s, target: %s)",
		borrowType,
		capabilityID,
		target,
	)
}

func AccountCapabilityController(borrowType string, capabilityID string) string {
	return fmt.Sprintf(
		"AccountCapabilityController(borrowType: Type<%s>(), capabilityID: %s)",
		borrowType,
		capabilityID,
	)
}
