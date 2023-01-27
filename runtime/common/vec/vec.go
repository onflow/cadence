package vec

import "golang.org/x/exp/constraints"

type Vec[T any] []T

// Convert a Vec to its underlying slice. This is a zero-cost coercion.
func (v Vec[T]) AsSlice() []T {
	return []T(v)
}

// Creates a Vec containing all arguments. If no arguments are passed, this function returns a nil slice.
// O(n)
func New[T any](items ...T) Vec[T] {
	return Vec[T](items)
}

// Creates a Vec with length 0 and the supplied capacity. Use this if you plan on inserting a known number of elements immediately.
// O(1)
func WithCapacity[T any](capacity int) Vec[T] {
	buf := make([]T, 0, capacity)
	return Vec[T](buf)
}

// Creates a Vec of length `len` with all elements initialized to the zero value for that type.
// O(n)
func Zeroed[T any](len int) Vec[T] {
	buf := make([]T, len, len)
	return Vec[T](buf)
}

// Appends a value to the end of the Vec.
// O(1)
func (v *Vec[T]) Push(val T) {
	*v = append(*v, val)
}

// Attempts to pop a value from the end of the Vec. `ok` is false if the Vec is nil or empty.
// O(1)
func (v *Vec[T]) Pop() (val T, ok bool) {
	buf := *v
	length := len(buf)
	if length == 0 {
		return
	}

	val = buf[length-1]
	ok = true
	*v = buf[:length-1]
	return
}

// Appends all elements of the other Vec to the end of this one.
// O(m)
func (v *Vec[T]) Concat(other Vec[T]) {
	*v = append(*v, other...)
}

// Returns a new Vec, containing all the elements that satisfy the given predicate.
// O(n)
func (v *Vec[T]) Filter(f func(T) bool) Vec[T] {
	res := WithCapacity[T](len(*v) / 2)
	for _, val := range *v {
		if f(val) {
			res.Push(val)
		}
	}
	return res
}

// Searches through the Vec to find the first element matching the given predicate. `ok` is false if no such element is found.
// O(n)
func (v *Vec[T]) Find(f func(T) bool) (val T, ok bool) {
	for _, x := range *v {
		if f(x) {
			return x, true
		}
	}
	return
}

// Returns a new Vec, containing the results of applying `f` to each value in `vec`.
// O(n)
func Map[A, B any](vec Vec[A], f func(A) B) Vec[B] {
	res := Zeroed[B](len(vec))
	for i, val := range vec {
		res[i] = f(val)
	}
	return res
}

// Returns a new Vec, containing the concatenated results of applying `f` to each value in `vec`.
func FlatMap[A, B any](vec Vec[A], f func(A) Vec[B]) Vec[B] {
	res := WithCapacity[B](len(vec))
	for _, valA := range vec {
		for _, valB := range f(valA) {
			res.Push(valB)
		}
	}
	return res
}

// Reduce `vec` from left to right using the supplied accumulator and combining function.
// O(n)
func Foldl[A, B any](vec Vec[A], acc B, combine func(B, A) B) B {
	for _, val := range vec {
		acc = combine(acc, val)
	}
	return acc
}

// Returns a `Vec` whose elements are the successive results of applying the `step` function to `seed`.
// Iteration stops the step function returns false for `ok`
// O(n)
func Unfold[A, B any](seed B, step func(B) (result A, next B, ok bool)) Vec[A] {
	res := New[A]()
	for {
		a, b, ok := step(seed)
		if !ok {
			break
		}

		res.Push(a)
		seed = b
	}

	return res
}

// Reduce `vec` from left to right using the supplied accumulator and combining function.
// Iteration stops when `combine` returns false for `ok`.
// O(n)
func FoldlWithBreak[A, B any](vec Vec[A], acc B, combine func(B, A) (result B, ok bool)) B {
	for _, val := range vec {
		res, ok := combine(acc, val)
		if !ok {
			break
		}
		acc = res
	}
	return acc
}

// Signals whether any value in `v` satisfies the given predicate. Short-circuits when a true value is found.
// O(n)
func (v *Vec[T]) Any(predicate func(T) bool) bool {
	for _, val := range *v {
		if predicate(val) {
			return true
		}
	}
	return false
}

// Signals whether all values in `v` satisfies the given predicate. Short-circuits when a false value is found.
// O(n)
func (v *Vec[T]) All(predicate func(T) bool) bool {
	for _, val := range *v {
		if !predicate(val) {
			return false
		}
	}
	return true
}

// Returns the number of items stored in `v`
// O(1)
func (v *Vec[T]) Length() int {
	return len(*v)
}

// Returns `true` if v is nil or has no items.
func (v *Vec[T]) IsEmpty() bool {
	return len(*v) == 0
}

type Numeric interface {
	constraints.Integer | constraints.Float | constraints.Complex
}

// Returns the numeric sum of all items in `vec`. If `vec` is empty, returns 0.
// O(n)
func Sum[T Numeric](vec Vec[T]) T {
	return Foldl(vec, 0, func(x, y T) T { return x + y })
}

// Returns the numeric product of all items in `vec`. If `vec` is empty, returns 1.
// O(n)
func Product[T Numeric](vec Vec[T]) T {
	return Foldl(vec, 1, func(x, y T) T { return x * y })
}

// Zip two vectors pairwise using the given combining function. The resulting Vec will have a length equal to the smaller of the two source vectors.
// O(min(n, m))
//
// Example:
//
//	xs := vec.New(1, 2, 3)
//	ys := vec.New(100, 200, 300, 400) // the last element is dropped
//	zs := vec.ZipWith(xs, ys, func(x, y int) int {
//	  return x + y
//	})
//	zs == Vec.new(101, 202, 303)
func ZipWith[A, B, C any](left Vec[A], right Vec[B], combine func(A, B) C) Vec[C] {
	minLen := len(left)
	if len(right) < minLen {
		minLen = len(right)
	}

	res := WithCapacity[C](minLen)
	for i := 0; i < minLen; i++ {
		res.Push(combine(left[i], right[i]))
	}

	return res
}
