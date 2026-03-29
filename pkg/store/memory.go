package store

// MemoryStore is the default in-process backing store.
// Keys are assumed to be pre-normalized by the caller (lowercased for
// variables/labels, quote-stripped for files). This avoids redundant
// string allocations on the hot path.
type MemoryStore struct {
	data [3]map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: [3]map[string]string{
			make(map[string]string),
			make(map[string]string),
			make(map[string]string),
		},
	}
}

func (m *MemoryStore) Get(ns Namespace, key string) (string, bool) {
	v, ok := m.data[ns][key]
	return v, ok
}

func (m *MemoryStore) Set(ns Namespace, key string, value string) {
	m.data[ns][key] = value
}

func (m *MemoryStore) Delete(ns Namespace, key string) {
	delete(m.data[ns], key)
}

func (m *MemoryStore) Exists(ns Namespace, key string) bool {
	_, ok := m.data[ns][key]
	return ok
}

func (m *MemoryStore) Keys(ns Namespace) []string {
	keys := make([]string, 0, len(m.data[ns]))
	for k := range m.data[ns] {
		keys = append(keys, k)
	}
	return keys
}

func (m *MemoryStore) Snapshot(ns Namespace) map[string]string {
	snap := make(map[string]string, len(m.data[ns]))
	for k, v := range m.data[ns] {
		snap[k] = v
	}
	return snap
}
