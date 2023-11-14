package permissions

import (
	"slices"
	"strings"
)

type Permission []string

func (p Permission) Has(perm string) bool {
	perms := strings.Split(perm, ".")
	var pc string
	for _, v := range perms {
		pc += v
		if slices.ContainsFunc(p, func(s string) bool {
			return s == pc || s == pc+".*"
		}) {
			return true
		}
		pc += "."
	}
	return false
}
