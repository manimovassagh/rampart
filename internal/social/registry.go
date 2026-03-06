package social

import "sort"

// Registry holds configured social providers indexed by name.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider to the registry, keyed by its Name().
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get returns the provider with the given name, if registered.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// Names returns a sorted list of all registered provider names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
