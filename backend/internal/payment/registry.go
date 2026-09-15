package payment

import (
	"fmt"
	"regexp"
)

var providerName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// Registry is filled once at startup and read-only afterwards, so reads need no lock.
type Registry struct {
	byName      map[string]Provider
	order       []string
	defaultName string
}

func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) error {
	name := p.Name()
	if !providerName.MatchString(name) {
		return fmt.Errorf("payment provider name %q must match %s", name, providerName)
	}
	if _, dup := r.byName[name]; dup {
		return fmt.Errorf("payment provider %q registered twice", name)
	}
	r.byName[name] = p
	r.order = append(r.order, name)
	return nil
}

func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.byName[name]
	return p, ok
}

// List returns providers in registration order.
func (r *Registry) List() []Provider {
	out := make([]Provider, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.byName[name])
	}
	return out
}

// SetDefault picks the provider used when a pay request names none; empty means the customer must choose.
func (r *Registry) SetDefault(name string) error {
	if name != "" {
		if _, ok := r.byName[name]; !ok {
			return fmt.Errorf("default payment provider %q is not enabled", name)
		}
	}
	r.defaultName = name
	return nil
}

func (r *Registry) Default() string { return r.defaultName }
