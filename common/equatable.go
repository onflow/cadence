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

package common

type Equatable[T any] interface {
	comparable
	Equal(other T) bool
}

func DeepEquals[T any, A, B Equatable[T]](source A, target B) bool {
	var emptyA A
	var emptyB B

	if source == emptyA {
		return target == emptyB
	} else if target == emptyB {
		return false
	}

	// Convert target to T to pass to source.Equal
	targetAsT := any(target).(T)
	return source.Equal(targetAsT)
}
