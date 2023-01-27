package vec

import "golang.org/x/exp/constraints"

type Vec[T any] []T

func (v Vec[T]) AsSlice() []T {
	return []T(v)
}

func New[T any](items ...T) Vec[T] {
	return Vec[T](items)
}

func WithCapacity[T any](capacity int) Vec[T] {
	buf := make([]T, 0, capacity)
	return Vec[T](buf)
}

func Zeroed[T any](len int) Vec[T] {
	buf := make([]T, len, len)
	return Vec[T](buf)
}

func (v *Vec[T]) Push(val T) {
	*v = append(*v, val)
}

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

func (v *Vec[T]) Concat(other Vec[T]) {
	*v = append(*v, other...)
}

func (v *Vec[T]) Filter(f func(T) bool) Vec[T] {
	res := WithCapacity[T](len(*v) / 2)
	for _, val := range *v {
		if f(val) {
			res.Push(val)
		}
	}
	return res
}

func (v *Vec[T]) Find(f func(T) bool) (val T, ok bool) {
	for _, x := range *v {
		if f(x) {
			return x, true
		}
	}
	return
}

func Map[A, B any](vec Vec[A], f func(A) B) Vec[B] {
	res := Zeroed[B](len(vec))
	for i, val := range vec {
		res[i] = f(val)
	}
	return res
}

func FlatMap[A, B any](vec Vec[A], f func(A) Vec[B]) Vec[B] {
	res := WithCapacity[B](len(vec))
	for _, valA := range vec {
		for _, valB := range f(valA) {
			res.Push(valB)
		}
	}
	return res
}

func Foldl[A, B any](vec Vec[A], acc B, combine func(B, A) B) B {
	for _, val := range vec {
		acc = combine(acc, val)
	}
	return acc
}

func Unfold[A, B any](seed B, next func(B) (A, B, bool)) Vec[A] {
	res := New[A]()
	for {
		a, b, ok := next(seed)
		if !ok {
			break
		}

		res.Push(a)
		seed = b
	}

	return res
}

func FoldlWithBreak[A, B any](vec Vec[A], acc B, combine func(B, A) (B, bool)) B {
	for _, val := range vec {
		res, ok := combine(acc, val)
		if !ok {
			break
		}
		acc = res
	}
	return acc
}

func (v *Vec[T]) Any(predicate func(T) bool) bool {
	for _, val := range *v {
		if predicate(val) {
			return true
		}
	}
	return false
}

func (v *Vec[T]) All(predicate func(T) bool) bool {
	for _, val := range *v {
		if !predicate(val) {
			return false
		}
	}
	return true
}

func (v *Vec[T]) Length() int {
	return len(*v)
}

func (v *Vec[T]) IsEmpty() bool {
	return len(*v) == 0
}

type Numeric interface {
	constraints.Integer | constraints.Float | constraints.Complex
}

func Sum[T Numeric](vec Vec[T]) T {
	return Foldl(vec, 0, func(x, y T) T { return x + y })
}

func Product[T Numeric](vec Vec[T]) T {
	return Foldl(vec, 1, func(x, y T) T { return x * y })
}

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
