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
	"path"
	"strings"

	"github.com/onflow/cadence/runtime/common"

	"github.com/onflow/cadence/languageserver/protocol"
)

func isPathLocation(location common.Location) bool {
	return locationToPath(location) != ""
}

func normalizePathLocation(base, relative common.Location) common.Location {
	basePath := locationToPath(base)
	relativePath := locationToPath(relative)

	if basePath == "" || relativePath == "" {
		return relative
	}

	normalizedPath := normalizePath(basePath, relativePath)

	return common.StringLocation(normalizedPath)
}

func normalizePath(basePath, relativePath string) string {
	if path.IsAbs(relativePath) {
		return relativePath
	}

	return path.Join(path.Dir(basePath), relativePath)
}

func locationToPath(location common.Location) string {
	stringLocation, ok := location.(common.StringLocation)
	if !ok {
		return ""
	}

	return string(stringLocation)
}

func uriToLocation(uri protocol.DocumentUri) common.StringLocation {
	return common.StringLocation(
		strings.TrimPrefix(string(uri), filePrefix),
	)
}
