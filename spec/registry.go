package spec

import (
	"fmt"
	"sync"
)

var (
	registry   = make(map[string]*Protocol)
	registryMu sync.RWMutex
)

// Register adds a protocol to the global registry. Returns an error if the
// name is already registered with a different protocol.
func Register(p *Protocol) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	if existing, ok := registry[p.Name]; ok {
		if existing == p {
			return nil // same pointer, idempotent
		}
		return fmt.Errorf("spec %q already registered", p.Name)
	}
	registry[p.Name] = p
	return nil
}

// MustRegister calls Register and panics on error.
func MustRegister(p *Protocol) {
	if err := Register(p); err != nil {
		panic(err)
	}
}

// Get returns the named protocol from the registry.
func Get(name string) (*Protocol, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("spec %q not found", name)
	}
	return p, nil
}

// MustGet returns the named protocol, panicking if not found.
func MustGet(name string) *Protocol {
	p, err := Get(name)
	if err != nil {
		panic(err)
	}
	return p
}

// List returns the names of all registered protocols.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
