package vec

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func wrong[T any](T) bool {
	return false
}

func TestNewVecIsNil(t *testing.T) {
	t.Parallel()
	require.Nil(t, New[int]())

	three := New(1, 2, 3)
	require.Equal(t, []int(three), []int{1, 2, 3})
}

func TestVecPush(t *testing.T) {
	t.Parallel()
	vec := New[int]()
	vec.Push(1)
	require.Equal(t, []int(vec), []int{1})
	vec.Push(2)
	require.Equal(t, []int(vec), []int{1, 2})
}

func TestVecZeroed(t *testing.T) {
	t.Parallel()
	vec := Zeroed[int](5)
	require.Equal(t, []int(vec), []int{0, 0, 0, 0, 0})
}

func TestVecPop(t *testing.T) {
	t.Parallel()
	t.Run("pop should remove from end", func(t *testing.T) {

		t.Parallel()
		v := Vec[int]([]int{1, 2, 3, 4, 5})
		stack := make([]int, 0, 5)
		for i := 0; i < 5; i++ {
			val, ok := v.Pop()
			require.True(t, ok)
			stack = append(stack, val)
		}
	})

	t.Run("pop should fail on empty vec", func(t *testing.T) {
		t.Parallel()

		vec := New[int]()
		zero, ok := vec.Pop()
		require.Zero(t, zero)
		require.False(t, ok)
	})
}

func TestVecConcat(t *testing.T) {
	t.Parallel()
	xs := New(1, 2, 3)
	ys := New(4, 5, 6)
	xs.Concat(ys)
	require.Equal(t, xs, New(1, 2, 3, 4, 5, 6))
}

func TestVecFilter(t *testing.T) {
	t.Parallel()

	xs := New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	evens := xs.Filter(func(x int) bool {
		return x%2 == 0
	})

	require.Equal(t, evens, New(2, 4, 6, 8, 10))

	odds := xs.Filter(func(x int) bool {
		return x%2 == 1
	})

	require.Equal(t, odds, New(1, 3, 5, 7, 9))

	none := xs.Filter(wrong[int])
	require.Empty(t, none)
}

func TestVecFind(t *testing.T) {
	t.Parallel()

	xs := New(0, 1, 2, 3, 4)
	for i := 0; i < len(xs); i++ {
		val, ok := xs.Find(func(x int) bool {
			return x == i
		})
		require.True(t, ok)
		require.Equal(t, i, val)
	}
	_, ok := xs.Find(wrong[int])

	require.False(t, ok)
}

func TestVecMap(t *testing.T) {
	t.Parallel()
	xs := New(1, 2, 3)
	ys := Map(xs, func(x int) string {
		return fmt.Sprint(x)
	})

	// should be pure
	require.Equal(t, New(1, 2, 3), xs)
	require.Equal(t, New("1", "2", "3"), ys)
}

func TestVecFlatMap(t *testing.T) {
	t.Parallel()

	xs := New(2, 3, 4)

	ys := FlatMap(xs, func(x int) Vec[int] {
		v := New[int]()
		for i := 0; i < x; i++ {
			v.Push(x)
		}
		return v
	})

	require.Equal(t, New(2, 3, 4), xs)

	expected := New(2, 2, 3, 3, 3, 4, 4, 4, 4)
	require.Equal(t, expected, ys)
}

func TestVecFoldl(t *testing.T) {
	t.Parallel()

	xs := New(1, 2, 3, 4, 5, 6)
	joined := Foldl(xs, "", func(acc string, x int) string {
		s := fmt.Sprint(x)
		return acc + s + s
	})

	// should be pure
	require.Equal(t, New(1, 2, 3, 4, 5, 6), xs)

	require.Equal(t, "112233445566", joined)

	id := Foldl(xs, New[int](), func(acc Vec[int], x int) Vec[int] {
		acc.Push(x)
		return acc
	})

	require.Equal(t, xs, id)
}

func TestVecUnfold(t *testing.T) {
	t.Parallel()

	xs := Unfold(0, func(prev int) (s string, next int, ok bool) {
		if prev >= 5 {
			return
		}

		return fmt.Sprint(prev), prev + 1, true
	})

	require.Equal(t, New("0", "1", "2", "3", "4"), xs)
}

func TestVecFoldlWithBreak(t *testing.T) {
	t.Parallel()

	makeExpected := func() Vec[int] { return New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10) }

	xs := makeExpected()
	ys := FoldlWithBreak(xs, New[int](), func(buf Vec[int], x int) (Vec[int], bool) {
		if x >= 5 {
			return nil, false
		}

		buf.Push(x)
		return buf, true
	})

	require.Equal(t, makeExpected(), xs)
	require.Equal(t, New(0, 1, 2, 3, 4), ys)
}

func TestVecBooleans(t *testing.T) {
	t.Parallel()

	makeVec := func() Vec[int] { return New(1, 2, 3, 4, 5) }
	original := makeVec()

	t.Run("test Vec.Any()", func(t *testing.T) {
		t.Parallel()

		xs := makeVec()
		ok := xs.Any(func(x int) bool {
			return x == 5
		})

		require.Equal(t, original, xs, "should not mutate")
		require.True(t, ok)

		require.False(t, xs.Any(wrong[int]))
	})

	t.Run("test Vec.All()", func(t *testing.T) {
		t.Parallel()

		xs := makeVec()
		ok := xs.All(func(x int) bool {
			return x != 6
		})

		require.Equal(t, original, xs, "should not mutate")
		require.True(t, ok)

		require.False(t, xs.All(wrong[int]))

		require.False(t, xs.All(func(x int) bool {
			return x == 6
		}))
	})
}

func TestVecLength(t *testing.T) {
	t.Parallel()

	xs := New(1, 2, 3)
	assertState := func(expectedLen int) {
		require.Equal(t, expectedLen, xs.Length())
		require.Equal(t, expectedLen == 0, xs.IsEmpty())
	}

	assertState(3)

	xs.Pop()
	xs.Pop()
	xs.Pop()
	assertState(0)

	xs.Push(1)
	assertState(1)

	xs = New[int]()
	assertState(0)

	xs = WithCapacity[int](100)
	assertState(0)
}

func TestVecNumericFolds(t *testing.T) {
	t.Parallel()

	makePrimes := func() Vec[int] {
		return New(2, 3, 5, 7, 11)
	}

	expected := makePrimes()
	xs := makePrimes()

	sum := Sum(xs)
	require.Equal(t, expected, xs, "should not mutate")
	require.Equal(t, 28, sum)

	product := Product(xs)
	require.Equal(t, expected, xs, "should not mutate")
	require.Equal(t, 2310, product)
}

func TestVecZipWith(t *testing.T) {
	t.Parallel()

	type Pair[A, B any] struct {
		car A
		cdr B
	}

	cons := func(s string, n int) Pair[string, int] {
		return Pair[string, int]{car: s, cdr: n}
	}

	makeVecs := func() (Vec[string], Vec[int]) {
		return New("a", "b", "c"), New(1, 2, 3)
	}
	expectedABC, expected123 := makeVecs()

	t.Run("zipWith should run pairwise", func(t *testing.T) {
		t.Parallel()

		vecABC, vec123 := makeVecs()

		zipped := ZipWith(vecABC, vec123, cons)

		require.Equal(t, expectedABC, vecABC, "should not mutate")
		require.Equal(t, expected123, vec123, "should not mutate")

		expected := New(cons("a", 1), cons("b", 2), cons("c", 3))
		require.Equal(t, expected, zipped)
	})

	t.Run("zipWith should terminate when one vec runs out", func(t *testing.T) {
		t.Parallel()
		expected := New(cons("a", 1), cons("b", 2), cons("c", 3))

		vecABC, vec123 := makeVecs()
		vec123.Push(4)
		vec123.Push(5)

		zipped := ZipWith(vecABC, vec123, cons)
		require.Equal(t, expected, zipped)

		vecABC.Push("d")
		expected.Push(cons("d", 4))

		zipped = ZipWith(vecABC, vec123, cons)
		require.Equal(t, expected, zipped)

		vecABC.Pop()
		vecABC.Pop() // bring it down to 2 items

		expected.Pop()
		expected.Pop() // bring it down to 2 items

		zipped = ZipWith(vecABC, vec123, cons)
		require.Equal(t, expected, zipped)
	})
}
