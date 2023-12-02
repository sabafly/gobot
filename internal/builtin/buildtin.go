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

func NonNilMap[T map[K]V, K comparable, V any](v T) T {
	if v != nil {
		return v
	} else {
		return make(T)
	}
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	} else {
		return v
	}
}
