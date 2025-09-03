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

// Package integer provides constants, as well as formatting, conversion,
// and checking functionality for Cadence integer number types
package integer

import "math/big"

var (
	Int128TypeMinIntBig = func() *big.Int {
		int128TypeMin := big.NewInt(-1)
		int128TypeMin.Lsh(int128TypeMin, 127)
		return int128TypeMin
	}()

	Int128TypeMaxIntBig = func() *big.Int {
		int128TypeMax := big.NewInt(1)
		int128TypeMax.Lsh(int128TypeMax, 127)
		int128TypeMax.Sub(int128TypeMax, big.NewInt(1))
		return int128TypeMax
	}()

	UInt128TypeMaxIntBig = func() *big.Int {
		uInt128TypeMax := big.NewInt(1)
		uInt128TypeMax.Lsh(uInt128TypeMax, 128)
		uInt128TypeMax.Sub(uInt128TypeMax, big.NewInt(1))
		return uInt128TypeMax

	}()
)
