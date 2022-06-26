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

package deps_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common/deps"
)

func TestNode_AllDependents(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, newNodeSet func() deps.NodeSet) {

		t.Parallel()

		//     A
		//    / \
		//   v   v
		//   D   B
		//   ^  /^
		//    \v  \
		//    C -> E

		a := deps.NewNode("a", deps.NewMapNodeSet)
		b := deps.NewNode("b", deps.NewMapNodeSet)
		c := deps.NewNode("c", deps.NewMapNodeSet)
		d := deps.NewNode("d", deps.NewMapNodeSet)
		e := deps.NewNode("e", deps.NewMapNodeSet)

		a.SetDependencies(d, b)
		b.SetDependencies(c, e)
		c.SetDependencies(d, e)

		aDependents, err := a.AllDependents()
		require.NoError(t, err)
		require.Equal(t, []*deps.Node{a}, aDependents)

		bDependents, err := b.AllDependents()
		require.NoError(t, err)
		require.Equal(t, []*deps.Node{a, b}, bDependents)

		cDependents, err := c.AllDependents()
		require.NoError(t, err)
		require.Equal(t, []*deps.Node{a, b, c}, cDependents)

		dDependents, err := d.AllDependents()
		require.NoError(t, err)
		require.Equal(t, []*deps.Node{a, b, c, d}, dDependents)

		eDependents, err := e.AllDependents()
		require.NoError(t, err)
		require.Equal(t, []*deps.Node{a, b, c, e}, eDependents)
	}

	t.Run("MapNodeSet", func(t *testing.T) {
		test(t, deps.NewMapNodeSet)
	})

	t.Run("OrderedNodeSet", func(t *testing.T) {
		test(t, deps.NewOrderedNodeSet)
	})

}

func TestNode_AllDependents_Circular(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, newNodeSet func() deps.NodeSet) {

		t.Parallel()

		//     A
		//    / \
		//   v   v
		//   D ->B
		//   ^  /^
		//    \v  \
		//    C -> E

		a := deps.NewNode("a", newNodeSet)
		b := deps.NewNode("b", newNodeSet)
		c := deps.NewNode("c", newNodeSet)
		d := deps.NewNode("d", newNodeSet)
		e := deps.NewNode("e", newNodeSet)

		a.SetDependencies(d, b)
		b.SetDependencies(c, e)
		c.SetDependencies(d, e)
		// NOTE: circular dependency
		d.SetDependencies(b)

		aDependents, err := a.AllDependents()
		require.NoError(t, err)
		require.Equal(t, []*deps.Node{a}, aDependents)

		bDependents, err := b.AllDependents()
		var bErr deps.CircularDependencyError
		require.ErrorAs(t, err, &bErr)
		require.Nil(t, bDependents)

		cDependents, err := c.AllDependents()
		var cErr deps.CircularDependencyError
		require.ErrorAs(t, err, &cErr)
		require.Nil(t, cDependents)

		dDependents, err := d.AllDependents()
		var dErr deps.CircularDependencyError
		require.ErrorAs(t, err, &dErr)
		require.Nil(t, dDependents)

		eDependents, err := e.AllDependents()
		var eErr deps.CircularDependencyError
		require.ErrorAs(t, err, &eErr)
		require.Nil(t, eDependents)
	}

	t.Run("MapNodeSet", func(t *testing.T) {
		test(t, deps.NewMapNodeSet)
	})

	t.Run("OrderedNodeSet", func(t *testing.T) {
		test(t, deps.NewOrderedNodeSet)
	})

}
