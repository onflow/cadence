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

package ccf

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCCFTypeID(t *testing.T) {
	testCases := []struct {
		name         string
		input        uint64
		encodedBytes []byte
	}{
		{name: "min", input: 0, encodedBytes: []byte{}},
		{name: "42", input: 42, encodedBytes: []byte{0x2a}},
		{name: "256", input: 256, encodedBytes: []byte{0x01, 0x00}},
		{name: "max", input: math.MaxUint64, encodedBytes: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create CCF type ID from uint64.
			ccfID := newCCFTypeIDFromUint64(tc.input)

			// Encode CCF type ID to bytes.
			encodedBytes := ccfID.Bytes()
			require.Equal(t, tc.encodedBytes, encodedBytes)

			// Decode CCF type ID from bytes.
			decodedCCFID := newCCFTypeID(encodedBytes)
			require.Equal(t, ccfTypeID(tc.input), decodedCCFID)

			// Compare decoded CCF type ID with original.
			require.True(t, ccfID.Equal(decodedCCFID))

			// Compare modified CCF type ID with original.
			if len(encodedBytes) == 0 {
				encodedBytes = []byte{0x00}
			}
			encodedBytes[0] = ^encodedBytes[0]
			require.False(t, ccfID.Equal(newCCFTypeID(encodedBytes)))
		})
	}
}

func TestCCFTypeIDByCadenceType(t *testing.T) {
	// Create ccfTypeIDByCadenceType map
	ccfIDs := make(ccfTypeIDByCadenceType)

	// Lookup non-existent CCF type ID.
	_, err := ccfIDs.id(simpleStructType())
	require.Error(t, err)

	// Add entry.
	ccfID := newCCFTypeIDFromUint64(1)
	ccfIDs[simpleStructType().ID()] = ccfID

	// Lookup existing CCF type ID.
	id, err := ccfIDs.id(simpleStructType())
	require.Equal(t, ccfID, id)
	require.NoError(t, err)
}

func TestCadenceTypeByCCFTypeID(t *testing.T) {
	cadenceTypes := newCadenceTypeByCCFTypeID()

	// Add new entry.
	newType := cadenceTypes.add(newCCFTypeIDFromUint64(0), simpleStructType())
	require.True(t, newType)

	// Add entry with duplicate CCF type ID.
	newType = cadenceTypes.add(newCCFTypeIDFromUint64(0), simpleStructType2())
	require.False(t, newType)

	// Lookup existing cadence type.
	typ, err := cadenceTypes.typ(newCCFTypeIDFromUint64(0))
	require.True(t, typ.Equal(simpleStructType()))
	require.False(t, typ.Equal(simpleStructType2()))
	require.NoError(t, err)

	// Lookup non-existent cadence type.
	typ, err = cadenceTypes.typ(newCCFTypeIDFromUint64(1))
	require.Nil(t, typ)
	require.Error(t, err)
}

func simpleStructType() *cadence.StructType {
	return &cadence.StructType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooStruct",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType{},
			},
		},
	}
}

func simpleStructType2() *cadence.StructType {
	return &cadence.StructType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooStruct2",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType{},
			},
			{
				Identifier: "b",
				Type:       cadence.StringType{},
			},
		},
	}
}
