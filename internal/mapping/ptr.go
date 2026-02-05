package mapping

func PtrOrNil[A any, B any](value *A, mappingFunc func(A) B) *B {
	if value == nil {
		return nil
	}
	b := mappingFunc(*value)
	return &b
}
