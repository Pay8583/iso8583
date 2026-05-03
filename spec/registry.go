package spec

import (
	"fmt"
	"sync"
)

var (
	registry   = make(map[string]*Spec)
	registryMu sync.RWMutex
)

// Register adds a spec to the global registry. Returns an error if the name
// is already registered.
func Register(s *Spec) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[s.Name]; ok {
		return fmt.Errorf("spec %q already registered", s.Name)
	}
	registry[s.Name] = s
	return nil
}

// MustRegister calls Register and panics on error.
func MustRegister(s *Spec) {
	if err := Register(s); err != nil {
		panic(err)
	}
}

// Get returns the named spec from the registry.
func Get(name string) (*Spec, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("spec %q not found", name)
	}
	return s, nil
}

// MustGet returns the named spec, panicking if not found.
func MustGet(name string) *Spec {
	s, err := Get(name)
	if err != nil {
		panic(err)
	}
	return s
}

// List returns the names of all registered specs.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
