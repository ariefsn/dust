package cleaner

import (
	"sort"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	cleaners map[string]Cleaner
}

func NewRegistry() *Registry {
	return &Registry{cleaners: make(map[string]Cleaner)}
}

func (r *Registry) Register(c Cleaner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleaners[c.ID()] = c
}

func (r *Registry) Get(id string) (Cleaner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.cleaners[id]
	return c, ok
}

func (r *Registry) All() []Cleaner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Cleaner, 0, len(r.cleaners))
	for _, c := range r.cleaners {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category() != out[j].Category() {
			return out[i].Category() < out[j].Category()
		}
		return out[i].ID() < out[j].ID()
	})
	return out
}

func (r *Registry) ByCategory() map[string][]Cleaner {
	out := make(map[string][]Cleaner)
	for _, c := range r.All() {
		out[c.Category()] = append(out[c.Category()], c)
	}
	return out
}
