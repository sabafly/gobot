package builtin

func Or[T any](ok bool, a, b T) T {
	if ok {
		return a
	} else {
		return b
	}
}

func Ptr[T any](v T) *T { return &v }
