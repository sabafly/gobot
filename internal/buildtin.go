package internal

func Or[T any](ok bool, a, b T) T {
	if ok {
		return a
	} else {
		return b
	}
}
