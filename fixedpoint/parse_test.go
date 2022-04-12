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

package fixedpoint

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFixedPoint(t *testing.T) {

	t.Parallel()

	type result struct {
		negative        bool
		unsignedInteger *big.Int
		fractional      *big.Int
		scale           uint
		err             error
	}

	for input, expectedResult := range map[string]result{
		"0.1":    {false, big.NewInt(0), big.NewInt(1), 1, nil},
		"-0.1":   {true, big.NewInt(0), big.NewInt(1), 1, nil},
		"1.0":    {false, big.NewInt(1), big.NewInt(0), 1, nil},
		"01.0":   {false, big.NewInt(1), big.NewInt(0), 1, nil},
		"-01.0":  {true, big.NewInt(1), big.NewInt(0), 1, nil},
		"1.23":   {false, big.NewInt(1), big.NewInt(23), 2, nil},
		"01.23":  {false, big.NewInt(1), big.NewInt(23), 2, nil},
		"-01.23": {true, big.NewInt(1), big.NewInt(23), 2, nil},
	} {
		t.Run(input, func(t *testing.T) {
			negative, unsignedInteger, fractional, parsedScale, err := parseFixedPoint(input)
			assert.Equal(t, negative, expectedResult.negative)
			assert.Equal(t, unsignedInteger, expectedResult.unsignedInteger)
			assert.Equal(t, fractional, expectedResult.fractional)
			assert.Equal(t, parsedScale, expectedResult.scale)
			assert.Equal(t, err, expectedResult.err)
		})
	}
}
