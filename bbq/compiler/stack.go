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

package compiler

type Stack[T any] struct {
	elements []T
}

func (s *Stack[T]) push(typ T) {
	s.elements = append(s.elements, typ)
}

func (s *Stack[T]) pop() T {
	lastIndex := len(s.elements) - 1
	top := s.elements[lastIndex]
	s.elements = s.elements[:lastIndex]
	return top
}

func (s *Stack[T]) top() T {
	lastIndex := len(s.elements) - 1
	return s.elements[lastIndex]
}

func (s *Stack[T]) bottom() T {
	return s.elements[0]
}

func (s *Stack[T]) isEmpty() bool {
	return len(s.elements) == 0
}
