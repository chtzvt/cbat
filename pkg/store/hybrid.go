package store

// HybridStore routes Variables and Labels to an in-memory store (per-session)
// and Files to a shared persistent store (SQLite).
type HybridStore struct {
	mem   *MemoryStore  // per-instance: variables, labels
	files Store         // shared: files (SQLite or any Store impl)
}

func NewHybridStore(sharedFiles Store) *HybridStore {
	return &HybridStore{
		mem:   NewMemoryStore(),
		files: sharedFiles,
	}
}

func (h *HybridStore) route(ns Namespace) Store {
	if ns == Files {
		return h.files
	}
	return h.mem
}

func (h *HybridStore) Get(ns Namespace, key string) (string, bool) {
	return h.route(ns).Get(ns, key)
}

func (h *HybridStore) Set(ns Namespace, key string, value string) {
	h.route(ns).Set(ns, key, value)
}

func (h *HybridStore) Delete(ns Namespace, key string) {
	h.route(ns).Delete(ns, key)
}

func (h *HybridStore) Exists(ns Namespace, key string) bool {
	return h.route(ns).Exists(ns, key)
}

func (h *HybridStore) Keys(ns Namespace) []string {
	return h.route(ns).Keys(ns)
}

func (h *HybridStore) Snapshot(ns Namespace) map[string]string {
	return h.route(ns).Snapshot(ns)
}
