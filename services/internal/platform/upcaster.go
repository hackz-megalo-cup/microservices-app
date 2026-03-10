package platform

import "encoding/json"

// Upcaster transforms event data from an older version to a newer one.
type Upcaster func(data json.RawMessage) json.RawMessage

// UpcasterRegistry holds upcasters keyed by event type.
type UpcasterRegistry struct {
	upcasters map[string][]Upcaster
}

// NewUpcasterRegistry creates an empty registry.
func NewUpcasterRegistry() *UpcasterRegistry {
	return &UpcasterRegistry{upcasters: make(map[string][]Upcaster)}
}

// Register adds an upcaster for a given event type.
// Upcasters are applied in registration order.
func (r *UpcasterRegistry) Register(eventType string, fn Upcaster) {
	r.upcasters[eventType] = append(r.upcasters[eventType], fn)
}

// Apply runs all registered upcasters for the given event type on the data.
func (r *UpcasterRegistry) Apply(eventType string, data json.RawMessage) json.RawMessage {
	if r == nil {
		return data
	}
	fns, ok := r.upcasters[eventType]
	if !ok {
		return data
	}
	for _, fn := range fns {
		data = fn(data)
	}
	return data
}
