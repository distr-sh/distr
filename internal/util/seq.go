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

// SeqCollect drains a fallible sequence into a slice, returning the first error
// encountered.
func SeqCollect[T any](seq iter.Seq2[T, error]) ([]T, error) {
	var values []T
	for value, err := range seq {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
