package generate

import (
	"fmt"
	"github.com/petermann-digital/quadlet-openrc/internal/quadlet"
	"sort"
	"strings"
)

func ValidateGraph(files []*quadlet.File) []error { return ValidateGraphWith(New(), files) }
func ValidateGraphWith(g *Generator, files []*quadlet.File) []error {
	known := map[string]bool{"podman": true}
	by := map[string]*quadlet.File{}
	for _, q := range files {
		s := g.serviceName(q)
		known[s] = true
		by[s] = q
	}
	var errs []error
	edges := map[string][]string{}
	for _, q := range files {
		n, _, a, _, _ := g.dependencies(q)
		deps := uniq(append(n, a...))
		edges[g.serviceName(q)] = deps
		for _, d := range n {
			if !known[d] {
				errs = append(errs, fmt.Errorf("%s: missing required Quadlet/OpenRC service %q", q.Path, d))
			}
		}
	}
	state := map[string]int{}
	var stack []string
	var visit func(string)
	visit = func(n string) {
		if state[n] == 2 {
			return
		}
		if state[n] == 1 {
			idx := 0
			for i, s := range stack {
				if s == n {
					idx = i
					break
				}
			}
			errs = append(errs, fmt.Errorf("dependency cycle: %s", strings.Join(append(stack[idx:], n), " -> ")))
			return
		}
		state[n] = 1
		stack = append(stack, n)
		for _, d := range edges[n] {
			if by[d] != nil {
				visit(d)
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = 2
	}
	var names []string
	for n := range by {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		visit(n)
	}
	return errs
}
