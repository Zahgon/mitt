package mitt

import "fmt"

func sprint(v any) string { return fmt.Sprint(v) }

// arity is the number of parameters the JavaScript original declares for a
// handler. It drives the truncation/padding rules reproduced below.
type arity uint8

const (
	arityTyped arity = iota
	arityWildcard
)

// Registration is a single entry in an emitter's handler list.
//
// It is a sealed union with exactly two implementations, [*Handler] and
// [*WildcardHandler], mirroring mitt's `Handler | WildcardHandler`.
//
// # Why handlers are handles and not plain funcs
//
// The TypeScript original identifies handlers by *reference*:
// `handlers.indexOf(handler)`. Go function values are not comparable, so a
// `func(E)` cannot be located in a slice. The usual workaround,
// reflect.ValueOf(fn).Pointer(), is incorrect: every closure produced from the
// same function literal shares one code pointer, so `Off` would remove an
// unrelated handler. Wrapping the function in a pointer type restores exactly
// the reference-identity semantics of JS.
//
// Keep the value returned by [NewHandler] / [Emitter.OnFunc] if you intend to
// call [Emitter.Off] later, just as you would keep a named function in JS.
type Registration[E any] interface {
	// invokeTyped calls the handler the way JS invokes a *type-matched*
	// handler: `handler(evt)`.
	invokeTyped(ev E)
	// invokeWildcard calls the handler the way JS invokes a *wildcard*
	// handler: `handler(type, evt)`.
	invokeWildcard(t EventType, ev E)
	// registration seals the interface and reports the handler's JS arity.
	registration() arity
}

// Handler is a registered single-argument event handler, equivalent to mitt's
// `Handler<T> = (event: T) => void`.
//
// Identity is pointer identity: two Handlers wrapping the same function are
// distinct registrations, and the same Handler registered twice fires twice.
type Handler[E any] struct {
	fn func(E)
}

// NewHandler wraps fn in an identity-comparable registration.
//
// A nil fn is permitted at registration time and panics when invoked, matching
// JS, where `on('foo', undefined)` succeeds and the TypeError surfaces at emit.
func NewHandler[E any](fn func(E)) *Handler[E] { return &Handler[E]{fn: fn} }

func (*Handler[E]) registration() arity { return arityTyped }

// Call invokes the handler directly. Provided for tests and manual dispatch.
func (h *Handler[E]) Call(ev E) { h.invokeTyped(ev) }

func (h *Handler[E]) invokeTyped(ev E) {
	if h.fn == nil {
		panic("mitt: handler is nil (equivalent to JS \"handler is not a function\")")
	}
	h.fn(ev)
}

// invokeWildcard reproduces JS arity truncation. A one-argument handler stored
// under '*' is called as `handler(type, evt)`; the extra argument is dropped and
// the handler actually receives the *event type*, not the payload.
func (h *Handler[E]) invokeWildcard(t EventType, _ E) {
	if h.fn == nil {
		panic("mitt: handler is nil (equivalent to JS \"handler is not a function\")")
	}
	h.fn(asPayload[E](t))
}

// WildcardHandler is a registered two-argument handler, equivalent to mitt's
// `WildcardHandler<T> = (type: keyof T, event: T[keyof T]) => void`.
type WildcardHandler[E any] struct {
	fn func(EventType, E)
}

// NewWildcardHandler wraps fn in an identity-comparable registration.
func NewWildcardHandler[E any](fn func(EventType, E)) *WildcardHandler[E] {
	return &WildcardHandler[E]{fn: fn}
}

func (*WildcardHandler[E]) registration() arity { return arityWildcard }

// Call invokes the handler directly. Provided for tests and manual dispatch.
func (h *WildcardHandler[E]) Call(t EventType, ev E) { h.invokeWildcard(t, ev) }

// invokeTyped reproduces JS arity padding. A two-argument wildcard handler
// reached through the *type* lookup is called as `handler(evt)`, so it receives
// the payload as its type argument and `undefined` (the zero value) as its
// event argument.
func (h *WildcardHandler[E]) invokeTyped(ev E) {
	if h.fn == nil {
		panic("mitt: handler is nil (equivalent to JS \"handler is not a function\")")
	}
	var zero E
	h.fn(asEventType(ev), zero)
}

func (h *WildcardHandler[E]) invokeWildcard(t EventType, ev E) {
	if h.fn == nil {
		panic("mitt: handler is nil (equivalent to JS \"handler is not a function\")")
	}
	h.fn(t, ev)
}

// asPayload converts an EventType into the payload type E for the JS case where
// a one-argument handler is invoked as `handler(type, evt)`.
//
// A Key unwraps to its underlying string when E can hold a string, which is
// what makes `emitter.on('*', fooHandler)` work in the TS type tests: the
// handler genuinely receives the event name.
func asPayload[E any](t EventType) E {
	if v, ok := any(t).(E); ok {
		return v
	}
	switch t.eventType() {
	case kindKey:
		if v, ok := any(string(t.(Key))).(E); ok {
			return v
		}
	case kindRaw:
		if v, ok := t.(*Raw).v.(E); ok {
			return v
		}
	}
	var zero E
	return zero
}

// asEventType converts a payload into the EventType position for the JS case
// where a two-argument handler is invoked as `handler(evt)`.
func asEventType[E any](ev E) EventType {
	if et, ok := any(ev).(EventType); ok {
		return et
	}
	if s, ok := any(ev).(string); ok {
		return Key(s)
	}
	return &Raw{v: any(ev)}
}

// isNilRegistration reports whether h is the Go equivalent of an omitted
// `handler` argument. It also catches typed-nil pointers stored in a non-nil
// interface, which is the classic Go footgun.
func isNilRegistration[E any](h Registration[E]) bool {
	if h == nil {
		return true
	}
	if h.registration() == arityTyped {
		return h.(*Handler[E]) == nil
	}
	return h.(*WildcardHandler[E]) == nil
}
