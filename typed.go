package mitt

import "fmt"

// OnTyped registers a handler that receives the payload already asserted to T.
//
// TypeScript gives each event key its own payload type through a mapped type
// (`Events[Key]`). Go generics cannot express a heterogeneous key-to-type map
// on a single value, so this helper recovers the ergonomics at the call site:
//
//	e := mitt.New[any]()
//	h := mitt.OnTyped[any, string](e, mitt.Key("foo"), func(s string) {
//		fmt.Println(len(s))
//	})
//
// The guarantee is weaker than TypeScript's: TS rejects a mismatched emit at
// compile time, whereas here a mismatched payload panics at dispatch with a
// descriptive message. A payload that is the zero value of E (the equivalent of
// emitting with no argument) is passed through as the zero value of T rather
// than panicking, matching `emit(type)` delivering `undefined`.
func OnTyped[E, T any](e *Emitter[E], t EventType, fn func(T)) *Handler[E] {
	return e.OnFunc(t, func(ev E) {
		fn(assertPayload[E, T](t, ev))
	})
}

// OnWildcardTyped registers a wildcard handler whose payload is asserted to T.
func OnWildcardTyped[E, T any](e *Emitter[E], fn func(EventType, T)) *WildcardHandler[E] {
	return e.OnWildcardFunc(func(t EventType, ev E) {
		fn(t, assertPayload[E, T](t, ev))
	})
}

// EmitTyped emits v as the payload for t, converting it to the emitter's
// payload type E.
//
// It panics if T is not assignable to E, which is a programming error rather
// than a runtime condition.
func EmitTyped[E, T any](e *Emitter[E], t EventType, v T) {
	ev, ok := any(v).(E)
	if !ok {
		var zeroE E
		panic(fmt.Sprintf(
			"mitt: cannot emit %T as payload type %T for event %v",
			v, zeroE, t,
		))
	}
	e.Emit(t, ev)
}

func assertPayload[E, T any](t EventType, ev E) T {
	if v, ok := any(ev).(T); ok {
		return v
	}
	var zero T
	if isZeroPayload(ev) {
		// `emit(type)` with no argument delivers undefined; deliver the zero
		// value rather than panicking.
		return zero
	}
	panic(fmt.Sprintf(
		"mitt: handler for event %v expected payload of type %T, got %T",
		t, zero, ev,
	))
}

func isZeroPayload[E any](ev E) bool {
	return any(ev) == nil
}
