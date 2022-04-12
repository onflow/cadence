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

package sema

// InterfaceSet

type InterfaceSet InterfaceTypeStructOrderedMap

func NewInterfaceSet() *InterfaceSet {
	return (*InterfaceSet)(NewInterfaceTypeStructOrderedMap())
}

func (s *InterfaceSet) IsSubsetOf(other *InterfaceSet) bool {
	orderedMap := (*InterfaceTypeStructOrderedMap)(s)

	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		interfaceType := pair.Key
		if !other.Includes(interfaceType) {
			return false
		}
	}

	return true
}

func (s *InterfaceSet) Includes(interfaceType *InterfaceType) bool {
	_, present := (*InterfaceTypeStructOrderedMap)(s).Get(interfaceType)
	return present
}

func (s *InterfaceSet) Add(interfaceType *InterfaceType) {
	orderedMap := (*InterfaceTypeStructOrderedMap)(s)
	orderedMap.Set(interfaceType, struct{}{})
}

func (s *InterfaceSet) ForEach(f func(*InterfaceType)) {
	orderedMap := (*InterfaceTypeStructOrderedMap)(s)
	orderedMap.Foreach(func(interfaceType *InterfaceType, _ struct{}) {
		f(interfaceType)
	})
}

func (s *InterfaceSet) Len() int {
	orderedMap := (*InterfaceTypeStructOrderedMap)(s)
	return orderedMap.Len()
}
