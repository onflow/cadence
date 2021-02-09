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

package sema

// MemberSet is set of members.
//
type MemberSet struct {
	Parent  *MemberSet
	members map[*Member]struct{}
}

// NewMemberSet returns an empty member set.
//
func NewMemberSet(parent *MemberSet) *MemberSet {
	return &MemberSet{
		members: map[*Member]struct{}{},
		Parent:  parent,
	}
}

// Add inserts a member into the set.
//
func (ms *MemberSet) Add(member *Member) {
	ms.members[member] = struct{}{}
}

// Contains returns true if the given member exists in the set.
//
func (ms *MemberSet) Contains(member *Member) bool {
	_, ok := ms.members[member]
	if ok {
		return true
	}

	if ms.Parent != nil {
		return ms.Parent.Contains(member)
	}

	return false
}

// ForEach calls the given function for each member.
// It can be used to iterate over all members of the set.
//
func (ms *MemberSet) ForEach(cb func(member *Member) error) error {
	memberSet := ms

	visited := map[*Member]bool{}

	for memberSet != nil {

		for member := range memberSet.members {
			if visited[member] {
				continue
			}

			err := cb(member)
			if err != nil {
				return err
			}

			visited[member] = true
		}

		memberSet = memberSet.Parent
	}

	return nil
}

// Intersection returns a new set containing all members that exist in this and the given set.
//
func (ms *MemberSet) Intersection(otherMS *MemberSet) *MemberSet {

	result := NewMemberSet(nil)

	_ = ms.ForEach(func(member *Member) error {
		if !otherMS.Contains(member) {
			return nil
		}

		result.Add(member)

		return nil
	})

	return result
}

// Clone returns a new child member set that contains all entries of this parent set.
// Changes to the returned set will only be applied in the returned set, not the parent.
//
func (ms *MemberSet) Clone() *MemberSet {
	return NewMemberSet(ms)
}
