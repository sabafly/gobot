package builtin

func Or[T any](ok bool, a, b T) T {
	if ok {
		return a
	}
	return b
}

func Ptr[T any](v T) *T { return &v }

func RequireNonNil[T any](v *T) T {
	if v == nil {
		panic("unexpected nil")
	}
	return *v
}

func NonNil[T any](v *T) T {
	if v != nil {
		return *v
	}
	var zero T
	return zero
}

func NonNilMap[T map[K]V, K comparable, V any](v T) T {
	if v != nil {
		return v
	}
	return make(T)
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func NonNilOrDefault[T any](v *T, def T) T {
	if v != nil {
		return *v
	}
	return def
}
