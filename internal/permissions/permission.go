package permissions

import (
	"strings"
)

type Permission map[string]bool

func (p Permission) Set(perm string, enabled bool) {
	if p == nil {
		p = make(Permission)
	}
	perm = strings.TrimSuffix(perm, ".*")
	p[perm] = enabled
}

func (p Permission) Remove(perm string) {
	delete(p, perm)
}

func (p Permission) Enabled(perm string) bool {
	if p == nil {
		return false
	}
	perms := strings.Split(perm, ".")
	var pc string
	var r bool
	for _, v := range perms {
		pc += v
		if a, ok := p[v]; ok {
			r = a
		}
		pc += "."
	}
	return r
}

func (p Permission) Disabled(perm string) bool {
	if p == nil {
		return false
	}
	perms := strings.Split(perm, ".")
	var pc string
	var r bool
	for _, v := range perms {
		pc += v
		if a, ok := p[v]; ok {
			r = !a
		}
		pc += "."
	}
	return r
}
