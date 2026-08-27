// Package mitt is a faithful Go port of developit/mitt, the tiny functional
// event emitter / pubsub.
//
//   - Useful: a wildcard "*" event type listens to all events
//   - Familiar: same names & ideas as Node's EventEmitter
//   - Minimal: on, off, emit and a publicly exposed handler map — nothing else
//
// The port is behaviour-preserving down to mitt's JavaScript-specific quirks.
// See DESIGN.md for the full list and the reasoning behind each API decision.
//
// Basic usage:
//
//	e := mitt.New[any]()
//
//	// listen to an event
//	foo := e.OnFunc(mitt.Key("foo"), func(ev any) { fmt.Println("foo", ev) })
//
//	// listen to all events
//	e.OnWildcardFunc(func(t mitt.EventType, ev any) { fmt.Println(t, ev) })
//
//	// fire an event
//	e.Emit(mitt.Key("foo"), map[string]string{"a": "b"})
//
//	// unlisten
//	e.Off(mitt.Key("foo"), foo)
//
//	// clearing all events
//	e.All().Clear()
package mitt

import "sync"

// Emitter is the Go equivalent of mitt's `Emitter<Events>`.
//
// The zero value is not usable; construct one with [New], [NewWith] or
// [NewSync].
//
// E is the payload type shared by every event on this emitter. TypeScript's
// per-key payload mapping (`Record<EventType, unknown>`) has no direct Go
// equivalent, so most callers use Emitter[any] and reach for [OnTyped] /
// [EmitTyped] where per-event static typing matters. See DESIGN.md,
// "Type-system parity gaps".
//
// An Emitter from [New] or [NewWith] is not safe for concurrent use, matching
// the single-threaded original. Use [NewSync] when you need goroutine safety.
type Emitter[E any] struct {
	all *HandlerMap[E]
	mu  *sync.RWMutex // nil unless created by NewSync
}

// New creates an empty emitter. Equivalent to `mitt()`.
func New[E any]() *Emitter[E] {
	return &Emitter[E]{all: NewHandlerMap[E]()}
}

// NewWith creates an emitter backed by an existing handler map, equivalent to
// `mitt(all)`. Handlers already present in the map are live immediately, and
// later mutations of the map are visible to the emitter.
//
// A nil map is treated as absent, matching `all = all || new Map()`.
func NewWith[E any](all *HandlerMap[E]) *Emitter[E] {
	if all == nil {
		all = NewHandlerMap[E]()
	}
	all.lazyInit()
	return &Emitter[E]{all: all, mu: all.mu}
}

// NewSync creates an emitter that is safe for concurrent use, with observable
// semantics identical to [New].
//
// Handler lists are snapshotted under the lock and dispatched with the lock
// released, so a handler may freely call On/Off/Emit on the same emitter
// without deadlocking — and, as in the original, those mutations do not affect
// the in-flight dispatch.
func NewSync[E any]() *Emitter[E] {
	mu := new(sync.RWMutex)
	m := NewHandlerMap[E]()
	m.mu = mu
	return &Emitter[E]{all: m, mu: mu}
}

// NewSyncWith creates a concurrency-safe emitter backed by an existing handler
// map. The map is adopted: it becomes guarded by the same lock, so direct
// mutation through [Emitter.All] is safe too.
//
// A nil map is treated as absent.
func NewSyncWith[E any](all *HandlerMap[E]) *Emitter[E] {
	if all == nil {
		all = NewHandlerMap[E]()
	}
	all.lazyInit()
	if all.mu == nil {
		all.mu = new(sync.RWMutex)
	}
	return &Emitter[E]{all: all, mu: all.mu}
}

// All returns the live handler map, equivalent to mitt's `emitter.all`.
//
// The map is not a copy: mutating it through [HandlerMap.Set],
// [HandlerMap.Delete] or [HandlerMap.Clear] changes what [Emitter.Emit]
// dispatches.
func (e *Emitter[E]) All() *HandlerMap[E] { return e.all }

// On registers a handler for the given event type, appending it to that type's
// handler list. Equivalent to `emitter.on(type, handler)`.
//
// Use [Wildcard] as the type to listen to every event.
//
// No de-duplication is performed. Registering the same handler twice registers
// it twice and it will be invoked twice per emit, matching Node's EventEmitter
// and mitt.
func (e *Emitter[E]) On(t EventType, h Registration[E]) {
	e.lock()
	defer e.unlock()
	if hs, ok := e.all.get(t); ok {
		e.all.set(t, append(hs, h))
		return
	}
	e.all.set(t, []Registration[E]{h})
}

// OnFunc wraps fn in a [Handler], registers it, and returns the handle so it
// can later be passed to [Emitter.Off].
//
// This is the ergonomic equivalent of keeping a named function reference in JS.
func (e *Emitter[E]) OnFunc(t EventType, fn func(E)) *Handler[E] {
	h := NewHandler(fn)
	e.On(t, h)
	return h
}

// OnWildcard registers a wildcard handler, equivalent to
// `emitter.on('*', handler)`.
func (e *Emitter[E]) OnWildcard(h *WildcardHandler[E]) { e.On(Wildcard, h) }

// OnWildcardFunc wraps fn in a [WildcardHandler], registers it under [Wildcard]
// and returns the handle.
func (e *Emitter[E]) OnWildcardFunc(fn func(EventType, E)) *WildcardHandler[E] {
	h := NewWildcardHandler(fn)
	e.OnWildcard(h)
	return h
}

// Off removes h from the handler list for t. Equivalent to
// `emitter.off(type, handler)`.
//
// Behaviour preserved from the original, all of it load-bearing:
//
//   - Only the *first* matching registration is removed. The same handler
//     registered twice needs two Off calls.
//   - Removing a handler that was never registered is a silent no-op. In JS
//     this falls out of `splice(indexOf(handler) >>> 0, 1)`: indexOf returns
//     -1, `-1 >>> 0` is 4294967295, and splicing past the end removes nothing.
//     A naive port removes the last element instead — this one does not.
//   - Calling Off on a type that has no entry does not create one.
//   - Passing a nil handler clears the type, exactly like omitting the
//     argument in JS. See [Emitter.OffAll].
func (e *Emitter[E]) Off(t EventType, h Registration[E]) {
	e.lock()
	defer e.unlock()

	hs, ok := e.all.get(t)
	if !ok {
		return
	}
	if isNilRegistration(h) {
		e.all.set(t, []Registration[E]{})
		return
	}

	idx := -1
	for i, cur := range hs {
		if cur == h {
			idx = i
			break
		}
	}
	if idx < 0 {
		// indexOf(...) >>> 0 lands out of range; splice removes nothing.
		return
	}
	e.all.set(t, append(hs[:idx:idx], hs[idx+1:]...))
}

// OffAll removes every handler for t, equivalent to `emitter.off(type)` with
// the handler argument omitted.
//
// The key is *not* deleted: it is left mapped to an empty list, so
// All().Has(t) still reports true afterwards. Use [HandlerMap.Delete] to remove
// the key outright.
//
// Calling OffAll on an unregistered type does nothing and does not create an
// entry.
func (e *Emitter[E]) OffAll(t EventType) { e.Off(t, nil) }

// Emit invokes every handler registered for t, then every [Wildcard] handler.
// Equivalent to `emitter.emit(type, evt)`.
//
// Behaviour preserved from the original:
//
//   - Dispatch is synchronous; Emit returns after the last handler returns.
//   - Type-matched handlers run first, in registration order, then wildcard
//     handlers, in registration order.
//   - Wildcard handlers receive the *original* event type, not [Wildcard].
//   - Each handler list is snapshotted immediately before it is dispatched, so
//     handlers added to or removed from the list being dispatched do not
//     affect the current emit.
//   - The wildcard list is looked up *after* the type-matched handlers have
//     run. A wildcard handler registered by a type-matched handler therefore
//     does fire during the same emit.
//   - Emitting an unregistered type is a silent no-op and does not create an
//     entry.
//   - Handlers are not isolated. A panicking handler propagates out of Emit
//     and the remaining handlers, wildcards included, do not run — matching
//     the original, which has no try/catch.
//
// Emitting [Wildcard] itself is not supported by mitt and behaves oddly by
// design; see [Emitter.Emit] notes in DESIGN.md.
func (e *Emitter[E]) Emit(t EventType, ev E) {
	if hs, ok := e.all.snapshot(t); ok {
		for _, h := range hs {
			h.invokeTyped(ev)
		}
	}
	if hs, ok := e.all.snapshot(Wildcard); ok {
		for _, h := range hs {
			h.invokeWildcard(t, ev)
		}
	}
}

// EmitVoid invokes handlers with the zero value as payload, equivalent to
// calling `emitter.emit(type)` with the event argument omitted, which passes
// `undefined`.
func (e *Emitter[E]) EmitVoid(t EventType) {
	var zero E
	e.Emit(t, zero)
}

func (e *Emitter[E]) lock() {
	if e.mu != nil {
		e.mu.Lock()
	}
}

func (e *Emitter[E]) unlock() {
	if e.mu != nil {
		e.mu.Unlock()
	}
}
