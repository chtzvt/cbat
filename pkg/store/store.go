package store

// Namespace distinguishes the logical key spaces in the VM.
type Namespace int

const (
	Variables Namespace = iota
	Files
	Labels
)

// Store is the pluggable persistence interface for the CBAT VM.
// All keys and values are strings. The VM handles type conversion.
type Store interface {
	Get(ns Namespace, key string) (string, bool)
	Set(ns Namespace, key string, value string)
	Delete(ns Namespace, key string)
	Exists(ns Namespace, key string) bool
	Keys(ns Namespace) []string
	Snapshot(ns Namespace) map[string]string
}
