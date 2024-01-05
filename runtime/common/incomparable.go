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

package common

// Incomparable (the zero-sized type [0]func()) makes the surrounding incomparable.
// It is crucial to ensure its placed at the beginning or middle of the surrounding struct,
// and NOT at the end of the struct, as otherwise the compiler will add padding bytes.
// See https://i.hsfzxjy.site/zst-at-the-rear-of-go-struct/ for more details
type Incomparable [0]func()
