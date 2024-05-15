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

package stdlib

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestSemaCheckPathLiteralForInternalStorageDomains(t *testing.T) {

	t.Parallel()

	internalStorageDomains := []string{
		InboxStorageDomain,
		AccountCapabilityStorageDomain,
		CapabilityControllerStorageDomain,
		PathCapabilityStorageDomain,
		CapabilityControllerTagStorageDomain,
	}

	test := func(domain string) {

		t.Run(domain, func(t *testing.T) {
			t.Parallel()

			_, err := sema.CheckPathLiteral(nil, domain, "test", nil, nil)
			var invalidPathDomainError *sema.InvalidPathDomainError
			require.ErrorAs(t, err, &invalidPathDomainError)
		})
	}

	for _, domain := range internalStorageDomains {
		test(domain)
	}
}
