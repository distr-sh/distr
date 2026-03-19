package util

// PtrTo creates a pointer to the given value
func PtrTo[T any](value T) *T {
	return new(value)
}

func PtrCopy[T any](ptr *T) *T {
	if ptr == nil {
		return nil
	}
	v := *ptr
	return &v
}

// PtrEq returns true iff both a and b are nil pointers or their dereferenced values are equal
func PtrEq[T comparable](a, b *T) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func PtrDerefOrDefault[T any](p *T) (v T) {
	if p != nil {
		v = *p
	}
	return
}
