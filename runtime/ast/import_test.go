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

package ast

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
)

func TestIdentifierLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := IdentifierLocation("test")

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "IdentifierLocation",
            "Identifier": "test"
        }
        `,
		string(actual),
	)
}

func TestStringLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := StringLocation("test")

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "StringLocation",
            "String": "test"
        }
        `,
		string(actual),
	)
}

func TestAddressLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := AddressLocation{
		Address: common.BytesToAddress([]byte{1}),
		Name:    "A",
	}

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "AddressLocation",
            "Address": "0x1",
            "Name": "A"
        }
        `,
		string(actual),
	)
}

func TestImportDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &ImportDeclaration{
		Identifiers: []Identifier{
			{
				Identifier: "foo",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Location:    StringLocation("test"),
		LocationPos: Position{Offset: 4, Line: 5, Column: 6},
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ImportDeclaration", 
            "Identifiers": [
                {
                    "Identifier": "foo",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                }
            ],
            "Location": {
                "Type": "StringLocation",
                "String": "test"
            },
            "LocationPos": {"Offset": 4, "Line": 5, "Column": 6},
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}
