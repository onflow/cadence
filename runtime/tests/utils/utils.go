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

package utils

import (
	"encoding/hex"
	errors2 "errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
)

// TestLocation is used as the default location for programs in tests.
const TestLocation = common.StringLocation("test")

// ImportedLocation is used as the default location for imported programs in tests.
const ImportedLocation = common.StringLocation("imported")

// AssertEqualWithDiff asserts that two objects are equal.
//
// If the objects are not equal, this function prints a human-readable diff.
func AssertEqualWithDiff(t *testing.T, expected, actual interface{}) {
	if !assert.Equal(t, expected, actual) {
		// the maximum levels of a struct to recurse into
		// this prevents infinite recursion from circular references
		deep.MaxDepth = 100

		diff := deep.Equal(expected, actual)

		if len(diff) != 0 {
			s := strings.Builder{}

			for i, d := range diff {
				if i == 0 {
					s.WriteString("diff    : ")
				} else {
					s.WriteString("          ")
				}

				s.WriteString(d)
				s.WriteString("\n")
			}

			t.Errorf("Not equal: \n"+
				"expected: %s\n"+
				"actual  : %s\n\n"+
				"%s", expected, actual, s.String(),
			)
		}
	}
}

func AsInterfaceType(name string, kind common.CompositeKind) string {
	switch kind {
	case common.CompositeKindResource, common.CompositeKindStructure:
		return fmt.Sprintf("{%s}", name)
	default:
		return name
	}
}

func DeploymentTransaction(name string, contract []byte) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.contracts.add(name: "%s", code: "%s".decodeHex())
              }
          }
        `,
		name,
		hex.EncodeToString(contract),
	))
}

func RemovalTransaction(name string) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.contracts.remove(name: "%s")
              }
          }
        `,
		name,
	))
}

func UpdateTransaction(name string, contract []byte) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.contracts.update__experimental(name: "%s", code: "%s".decodeHex())
              }
          }
        `,
		name,
		hex.EncodeToString(contract),
	))
}

// TODO: switch to require.ErrorAs once released:
// https://github.com/stretchr/testify/commit/95a9d909e98735cd8211dfc5cbbb6b8b0b665915
func RequireErrorAs(t *testing.T, err error, target interface{}) {
	require.True(
		t,
		errors2.As(err, target),
		"error chain must contain a %T",
		target,
	)
}
