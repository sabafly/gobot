package builtin

func Or[T any](ok bool, a, b T) T {
	if ok {
		return a
	} else {
		return b
	}
}

func Ptr[T any](v T) *T { return &v }

func NonNil[T any](v *T) T {
	if v != nil {
		return *v
	} else {
		return *new(T)
	}
}
