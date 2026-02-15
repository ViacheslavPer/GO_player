package storage

import "testing"

// Storage is an interface with no methods in MVP. This test ensures the interface compiles
// and can be satisfied by a concrete type for future use.
func TestStorage_InterfaceExists(t *testing.T) {
	type stub struct{}
	var _ Storage = (*stub)(nil)
}
