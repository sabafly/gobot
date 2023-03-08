package handler

type Check[T any] func(ctx T) bool

func (c Check[T]) Or(check Check[T]) Check[T] {
	return func(ctx T) bool {
		return c(ctx) || check(ctx)
	}
}

func (c Check[T]) And(check Check[T]) Check[T] {
	return func(ctx T) bool {
		return c(ctx) && check(ctx)
	}
}

func CheckAnyOf[T any](checks ...Check[T]) Check[T] {
	var check Check[T]
	for _, c := range checks {
		check = check.Or(c)
	}
	return check
}

func CheckAllOf[T any](checks ...Check[T]) Check[T] {
	var check Check[T]
	for _, c := range checks {
		check = check.And(c)
	}
	return check
}
