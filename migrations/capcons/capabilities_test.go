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

package capcons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
)

func TestCapabilitiesIteration(t *testing.T) {
	t.Parallel()

	caps := AccountCapabilities{}

	caps.Record(
		interpreter.NewUnmeteredPathValue(common.PathDomainPublic, "b"),
		nil,
		interpreter.StorageKey{},
		nil,
	)

	caps.Record(
		interpreter.NewUnmeteredPathValue(common.PathDomainPublic, "a"),
		nil,
		interpreter.StorageKey{},
		nil,
	)

	caps.Record(
		interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "c"),
		nil,
		interpreter.StorageKey{},
		nil,
	)

	caps.Record(
		interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "a"),
		nil,
		interpreter.StorageKey{},
		nil,
	)

	caps.Record(
		interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "b"),
		nil,
		interpreter.StorageKey{},
		nil,
	)

	require.False(t, caps.sorted)

	var paths []interpreter.PathValue
	caps.ForEachSorted(func(capability AccountCapability) bool {
		paths = append(paths, capability.TargetPath)
		return true
	})

	require.True(t, caps.sorted)

	assert.Equal(
		t,
		[]interpreter.PathValue{
			interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "a"),
			interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "b"),
			interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "c"),
			interpreter.NewUnmeteredPathValue(common.PathDomainPublic, "a"),
			interpreter.NewUnmeteredPathValue(common.PathDomainPublic, "b"),
		},
		paths,
	)

	caps.Record(
		interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "aa"),
		nil,
		interpreter.StorageKey{},
		nil,
	)

	require.False(t, caps.sorted)

	paths = make([]interpreter.PathValue, 0)
	caps.ForEachSorted(func(capability AccountCapability) bool {
		paths = append(paths, capability.TargetPath)
		return true
	})

	require.True(t, caps.sorted)

	assert.Equal(
		t,
		[]interpreter.PathValue{
			interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "a"),
			interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "aa"),
			interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "b"),
			interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "c"),
			interpreter.NewUnmeteredPathValue(common.PathDomainPublic, "a"),
			interpreter.NewUnmeteredPathValue(common.PathDomainPublic, "b"),
		},
		paths,
	)
}
