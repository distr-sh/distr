package util

import "iter"

func SeqFilter[T any](seq iter.Seq[T], p func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if p(v) && !yield(v) {
				break
			}
		}
	}
}

func SeqLen[T any](seq iter.Seq[T]) (l int) {
	for range seq {
		l++
	}
	return
}
