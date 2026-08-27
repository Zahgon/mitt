package mitt

import "sync"

// HandlerMap is the Go equivalent of mitt's `all` — a `Map` of event names to
// handler lists, exposed publicly and intended to be read *and written* by
// callers.
//
// It preserves insertion order, because JavaScript's Map does and mitt's own
// test suite inspects `all` directly. A plain Go map randomises iteration order
// and would not be equivalent.
//
// Direct mutation is fully supported and is visible to [Emitter.Emit]:
//
//	m := mitt.NewHandlerMap[any]()
//	m.Set(mitt.Wildcard, []mitt.Registration[any]{star})
//	e := mitt.NewWith(m)
//	e.Emit(mitt.Key("foo"), payload) // star fires
type HandlerMap[E any] struct {
	mu    *sync.RWMutex // nil unless created by NewSync
	keys  []EventType
	index map[EventType]int
	vals  [][]Registration[E]
}

// NewHandlerMap creates an empty, insertion-ordered handler map. Equivalent to
// `new Map()`.
func NewHandlerMap[E any]() *HandlerMap[E] {
	return &HandlerMap[E]{index: make(map[EventType]int)}
}

func (m *HandlerMap[E]) lazyInit() {
	if m.index == nil {
		m.index = make(map[EventType]int)
	}
}

func (m *HandlerMap[E]) rlock() {
	if m.mu != nil {
		m.mu.RLock()
	}
}

func (m *HandlerMap[E]) runlock() {
	if m.mu != nil {
		m.mu.RUnlock()
	}
}

func (m *HandlerMap[E]) lock() {
	if m.mu != nil {
		m.mu.Lock()
	}
}

func (m *HandlerMap[E]) unlock() {
	if m.mu != nil {
		m.mu.Unlock()
	}
}

// Get returns the handler list registered for t. Equivalent to `map.get(type)`;
// ok is false when the key is absent, matching an `undefined` return.
//
// A present-but-empty list (ok == true, len == 0) is meaningful: that is the
// state left behind by [Emitter.OffAll].
func (m *HandlerMap[E]) Get(t EventType) (handlers []Registration[E], ok bool) {
	m.rlock()
	defer m.runlock()
	return m.get(t)
}

func (m *HandlerMap[E]) get(t EventType) ([]Registration[E], bool) {
	i, ok := m.index[t]
	if !ok {
		return nil, false
	}
	return m.vals[i], true
}

// Set stores handlers under t, equivalent to `map.set(type, handlers)`.
// A new key is appended at the end of the iteration order; an existing key
// keeps its original position.
func (m *HandlerMap[E]) Set(t EventType, handlers []Registration[E]) {
	m.lock()
	defer m.unlock()
	m.set(t, handlers)
}

func (m *HandlerMap[E]) set(t EventType, handlers []Registration[E]) {
	m.lazyInit()
	if i, ok := m.index[t]; ok {
		m.vals[i] = handlers
		return
	}
	m.index[t] = len(m.keys)
	m.keys = append(m.keys, t)
	m.vals = append(m.vals, handlers)
}

// Has reports whether t is present, equivalent to `map.has(type)`. It returns
// true even when the stored list is empty.
func (m *HandlerMap[E]) Has(t EventType) bool {
	m.rlock()
	defer m.runlock()
	_, ok := m.index[t]
	return ok
}

// Delete removes t entirely, equivalent to `map.delete(type)`. It reports
// whether the key was present.
//
// Note this is *not* what off(type) does — see [Emitter.OffAll].
func (m *HandlerMap[E]) Delete(t EventType) bool {
	m.lock()
	defer m.unlock()
	i, ok := m.index[t]
	if !ok {
		return false
	}
	m.keys = append(m.keys[:i], m.keys[i+1:]...)
	m.vals = append(m.vals[:i], m.vals[i+1:]...)
	delete(m.index, t)
	for j := i; j < len(m.keys); j++ {
		m.index[m.keys[j]] = j
	}
	return true
}

// Len returns the number of registered event types, equivalent to `map.size`.
func (m *HandlerMap[E]) Len() int {
	m.rlock()
	defer m.runlock()
	return len(m.keys)
}

// Clear removes every entry, equivalent to `emitter.all.clear()` as shown in
// mitt's README.
func (m *HandlerMap[E]) Clear() {
	m.lock()
	defer m.unlock()
	m.keys = nil
	m.vals = nil
	m.index = make(map[EventType]int)
}

// Keys returns the event types in insertion order, equivalent to
// `[...map.keys()]`.
func (m *HandlerMap[E]) Keys() []EventType {
	m.rlock()
	defer m.runlock()
	out := make([]EventType, len(m.keys))
	copy(out, m.keys)
	return out
}

// Range iterates entries in insertion order, equivalent to `map.forEach`.
// Returning false from fn stops the iteration.
//
// Iteration is performed over a snapshot, so fn may safely mutate the map.
func (m *HandlerMap[E]) Range(fn func(t EventType, handlers []Registration[E]) bool) {
	m.rlock()
	keys := make([]EventType, len(m.keys))
	copy(keys, m.keys)
	vals := make([][]Registration[E], len(m.vals))
	copy(vals, m.vals)
	m.runlock()

	for i, k := range keys {
		if !fn(k, vals[i]) {
			return
		}
	}
}

// snapshot returns a defensive copy of the handler list for t.
//
// mitt calls `.slice()` before dispatching so that handlers registered or
// removed *during* an emit do not affect the in-flight dispatch. This is the
// Go equivalent.
func (m *HandlerMap[E]) snapshot(t EventType) ([]Registration[E], bool) {
	m.rlock()
	defer m.runlock()
	hs, ok := m.get(t)
	if !ok {
		return nil, false
	}
	out := make([]Registration[E], len(hs))
	copy(out, hs)
	return out, true
}
