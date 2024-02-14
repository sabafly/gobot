package ratelimit

import (
	"time"
)

type Rule struct {
	Limit int
	Unit  time.Duration
}

// CheckLimit はセーフならtrueを返す
func CheckLimit(t []time.Time, r []Rule) ([]time.Time, bool) {
	m := 0
	for _, v := range r {
		if !check(t, v.Limit, v.Unit) {
			return t, false
		}
		m = max(v.Limit, m)
	}
	t = append([]time.Time{time.Now()}, (t)[0:min(m, len(t))]...)
	return t, true
}

func check(times []time.Time, limit int, unit time.Duration) bool {
	return len(times) < limit || time.Since(times[limit-1]) >= unit
}
