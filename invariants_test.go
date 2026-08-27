package mitt_test

import (
	"fmt"
	"reflect"
	"testing"

	mitt "github.com/developit/mitt-go"
)

// Each test below pins a JavaScript-specific behaviour of the original that a
// naive port gets wrong. Every expectation was verified by executing the
// upstream src/index.ts under Node before being encoded here; see DESIGN.md.

// Invariant 1: `splice(indexOf(h) >>> 0, 1)` with an absent handler splices at
// index 4294967295 and removes nothing. A naive port removes the last element.
func TestOffUnregisteredHandlerIsNoop(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	a := e.OnFunc(mitt.Key("foo"), func(any) {})
	b := mitt.NewHandler[any](func(any) {})

	e.Off(mitt.Key("foo"), b)

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), a)
}

func TestOffUnregisteredHandlerDoesNotRemoveLast(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	a := e.OnFunc(mitt.Key("foo"), func(any) {})
	b := e.OnFunc(mitt.Key("foo"), func(any) {})
	stranger := mitt.NewHandler[any](func(any) {})

	e.Off(mitt.Key("foo"), stranger)

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), a, b)
}

// Invariant 4: the `if (handlers)` guard means off() on an absent type never
// creates an entry.
func TestOffUnknownTypeDoesNotCreateEntry(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	e.OffAll(mitt.Key("never-registered"))
	e.Off(mitt.Key("also-never"), mitt.NewHandler[any](func(any) {}))

	if m.Has(mitt.Key("never-registered")) || m.Has(mitt.Key("also-never")) {
		t.Fatal("off must not create handler-map entries")
	}
	if m.Len() != 0 {
		t.Fatalf("Len = %d, want 0", m.Len())
	}
}

// Invariant 6a: the type-matched list is snapshotted before dispatch, so a
// handler registered during the emit does not fire in that emit.
func TestEmitSnapshotsHandlersAdditionDuringDispatch(t *testing.T) {
	e := mitt.New[any]()
	var order []string

	e.OnFunc(mitt.Key("foo"), func(any) {
		order = append(order, "a")
		e.OnFunc(mitt.Key("foo"), func(any) { order = append(order, "b") })
	})

	e.Emit(mitt.Key("foo"), 1)

	if !reflect.DeepEqual(order, []string{"a"}) {
		t.Fatalf("order = %v, want [a]", order)
	}

	e.Emit(mitt.Key("foo"), 1)
	if !reflect.DeepEqual(order, []string{"a", "a", "b"}) {
		t.Fatalf("second emit order = %v, want [a a b]", order)
	}
}

// Invariant 6b: removal during dispatch also does not affect the in-flight
// emit — the already-snapshotted later handler still runs.
func TestEmitSnapshotsHandlersRemovalDuringDispatch(t *testing.T) {
	e := mitt.New[any]()
	var order []string

	second := mitt.NewHandler[any](func(any) { order = append(order, "second") })
	e.OnFunc(mitt.Key("foo"), func(any) {
		order = append(order, "first")
		e.Off(mitt.Key("foo"), second)
	})
	e.On(mitt.Key("foo"), second)

	e.Emit(mitt.Key("foo"), 1)

	if !reflect.DeepEqual(order, []string{"first", "second"}) {
		t.Fatalf("order = %v, want [first second]", order)
	}

	e.Emit(mitt.Key("foo"), 1)
	if !reflect.DeepEqual(order, []string{"first", "second", "first"}) {
		t.Fatalf("second emit order = %v, want [first second first]", order)
	}
}

// Invariant: the wildcard list is fetched *after* the type-matched handlers
// have run, so a wildcard handler registered mid-dispatch does fire.
func TestWildcardListLookedUpAfterTypedDispatch(t *testing.T) {
	e := mitt.New[any]()
	var order []string

	e.OnFunc(mitt.Key("foo"), func(any) {
		order = append(order, "typed")
		e.OnWildcardFunc(func(mitt.EventType, any) {
			order = append(order, "late-wildcard")
		})
	})

	e.Emit(mitt.Key("foo"), 1)

	if !reflect.DeepEqual(order, []string{"typed", "late-wildcard"}) {
		t.Fatalf("order = %v, want [typed late-wildcard]", order)
	}
}

// Invariant 7: type-matched handlers run before wildcard handlers, each group
// in registration order.
func TestDispatchOrder(t *testing.T) {
	e := mitt.New[any]()
	var order []string

	e.OnWildcardFunc(func(mitt.EventType, any) { order = append(order, "w1") })
	e.OnFunc(mitt.Key("foo"), func(any) { order = append(order, "t1") })
	e.OnWildcardFunc(func(mitt.EventType, any) { order = append(order, "w2") })
	e.OnFunc(mitt.Key("foo"), func(any) { order = append(order, "t2") })

	e.Emit(mitt.Key("foo"), nil)

	if !reflect.DeepEqual(order, []string{"t1", "t2", "w1", "w2"}) {
		t.Fatalf("order = %v, want [t1 t2 w1 w2]", order)
	}
}

// Invariant 8: wildcard handlers receive the original event type, never '*'.
func TestWildcardReceivesOriginalType(t *testing.T) {
	e := mitt.New[any]()
	sym := mitt.Sym("s")
	var got []mitt.EventType

	e.OnWildcardFunc(func(t mitt.EventType, _ any) { got = append(got, t) })

	e.Emit(mitt.Key("foo"), nil)
	e.Emit(sym, nil)

	if len(got) != 2 || got[0] != mitt.Key("foo") || got[1] != mitt.EventType(sym) {
		t.Fatalf("got = %v, want [foo Symbol(s)]", got)
	}
}

// Invariant 9: emitting '*' resolves the same list twice — once through the
// type lookup and once through the wildcard lookup — so wildcard handlers fire
// twice with different arguments. Node output for the equivalent program:
//
//	[[{"p":1}, undefined], ["*", {"p":1}]]
//
// mitt documents that manually firing '*' is unsupported; this pins the
// behaviour rather than endorsing it.
func TestEmitWildcardKeyDoubleInvokes(t *testing.T) {
	e := mitt.New[any]()
	type got struct {
		t  mitt.EventType
		ev any
	}
	var calls []got

	e.OnWildcardFunc(func(tp mitt.EventType, ev any) {
		calls = append(calls, got{tp, ev})
	})

	payload := map[string]int{"p": 1}
	e.Emit(mitt.Wildcard, payload)

	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}

	raw, ok := calls[0].t.(*mitt.Raw)
	if !ok {
		t.Fatalf("first call type = %#v, want *Raw boxing the payload", calls[0].t)
	}
	if !reflect.DeepEqual(raw.Value(), any(payload)) {
		t.Errorf("boxed value = %#v, want %#v", raw.Value(), payload)
	}
	if calls[0].ev != nil {
		t.Errorf("first call event = %#v, want nil (JS undefined)", calls[0].ev)
	}

	if calls[1].t != mitt.EventType(mitt.Wildcard) {
		t.Errorf("second call type = %#v, want Wildcard", calls[1].t)
	}
	if !reflect.DeepEqual(calls[1].ev, any(payload)) {
		t.Errorf("second call event = %#v, want payload", calls[1].ev)
	}
}

// Invariant 8: the Raw box is an implementation detail of the Go type system
// and must not be visible when the event type is rendered.
//
// JavaScript has no box at all: `emit('*', 42)` invokes the wildcard handler as
// `handler(42)`, so its `type` parameter *is* the payload and `String(type)`
// yields "42". Rendering the Go equivalent as `Raw("42")` would leak a wrapper
// the original has no counterpart for, so Raw.String must be transparent.
//
// Verified against upstream src/index.ts under Node v26.7.0, printing
// `String(type)` for a wildcard handler after `emit('*', payload)`:
//
//	42 -> "42"   1.5 -> "1.5"   true -> "true"
//
// Only scalars are pinned. Composites are rendered by each language's own
// stringifier ([1,2] is "1,2" in JS and "[1 2]" in Go, {p:1} is
// "[object Object]" versus "map[p:1]"), a general cross-language difference
// that transparency neither causes nor can repair.
func TestRawRendersTransparently(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload any
		want    string
	}{
		{"int", 42, "42"},
		{"float", 1.5, "1.5"},
		{"bool", true, "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := mitt.New[any]()
			var boxed mitt.EventType
			e.OnWildcardFunc(func(tp mitt.EventType, _ any) {
				if boxed == nil {
					boxed = tp
				}
			})
			e.Emit(mitt.Wildcard, tc.payload)

			if _, ok := boxed.(*mitt.Raw); !ok {
				t.Fatalf("type position = %#v, want *Raw", boxed)
			}
			if got := fmt.Sprint(boxed); got != tc.want {
				t.Errorf("fmt.Sprint(type) = %q, want %q", got, tc.want)
			}
		})
	}
}

// Invariant: a one-argument handler registered under '*' is invoked as
// handler(type, evt); JS drops the extra argument, so it receives the event
// *type*. This is what makes `emitter.on('*', fooHandler)` typecheck in the
// upstream type tests.
func TestSingleArgHandlerOnWildcardReceivesType(t *testing.T) {
	e := mitt.New[any]()
	var got []any

	e.On(mitt.Wildcard, mitt.NewHandler[any](func(v any) { got = append(got, v) }))
	e.Emit(mitt.Key("foo"), "PAYLOAD")

	if len(got) != 1 {
		t.Fatalf("calls = %d, want 1", len(got))
	}
	if got[0] != mitt.EventType(mitt.Key("foo")) {
		t.Fatalf("received %#v, want the event type Key(\"foo\")", got[0])
	}
}

func TestSingleArgStringHandlerOnWildcardReceivesTypeAsString(t *testing.T) {
	e := mitt.New[string]()
	var got []string

	e.On(mitt.Wildcard, mitt.NewHandler[string](func(v string) { got = append(got, v) }))
	e.Emit(mitt.Key("foo"), "PAYLOAD")

	if !reflect.DeepEqual(got, []string{"foo"}) {
		t.Fatalf("got = %v, want [foo]", got)
	}
}

// Invariant 14: emit(type) with no payload delivers undefined.
func TestEmitVoidDeliversZeroValue(t *testing.T) {
	e := mitt.New[any]()
	var got []any
	e.OnFunc(mitt.Key("foo"), func(ev any) { got = append(got, ev) })

	e.EmitVoid(mitt.Key("foo"))

	if len(got) != 1 || got[0] != nil {
		t.Fatalf("got = %#v, want [nil]", got)
	}
}

// Invariant 15/16: dispatch is synchronous and unguarded. A panicking handler
// propagates out of Emit and aborts the rest of the dispatch, wildcards
// included, because the original has no try/catch.
func TestPanicInHandlerAbortsDispatch(t *testing.T) {
	e := mitt.New[any]()
	var order []string

	e.OnFunc(mitt.Key("foo"), func(any) { panic("boom") })
	e.OnFunc(mitt.Key("foo"), func(any) { order = append(order, "after") })
	e.OnWildcardFunc(func(mitt.EventType, any) { order = append(order, "wild") })

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic to propagate out of Emit")
			} else {
				order = append(order, "recovered")
			}
		}()
		e.Emit(mitt.Key("foo"), 1)
	}()

	if !reflect.DeepEqual(order, []string{"recovered"}) {
		t.Fatalf("order = %v, want [recovered]", order)
	}
}

func TestNilHandlerPanicsOnInvoke(t *testing.T) {
	e := mitt.New[any]()
	e.On(mitt.Key("foo"), mitt.NewHandler[any](nil))

	defer func() {
		if recover() == nil {
			t.Error("expected panic from nil handler")
		}
	}()
	e.Emit(mitt.Key("foo"), 1)
}

func TestNilWildcardHandlerPanicsOnInvoke(t *testing.T) {
	e := mitt.New[any]()
	e.OnWildcard(mitt.NewWildcardHandler[any](nil))

	defer func() {
		if recover() == nil {
			t.Error("expected panic from nil wildcard handler")
		}
	}()
	e.Emit(mitt.Key("foo"), 1)
}

func TestNilWildcardHandlerPanicsOnTypedInvoke(t *testing.T) {
	e := mitt.New[any]()
	e.On(mitt.Key("foo"), mitt.NewWildcardHandler[any](nil))

	defer func() {
		if recover() == nil {
			t.Error("expected panic from nil wildcard handler on typed dispatch")
		}
	}()
	e.Emit(mitt.Key("foo"), 1)
}

// Off with a typed-nil pointer must behave like an omitted argument, not like a
// handler that happens to be missing.
func TestOffWithTypedNilClearsType(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)
	e.OnFunc(mitt.Key("foo"), func(any) {})

	e.Off(mitt.Key("foo"), (*mitt.Handler[any])(nil))

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")))
	if !m.Has(mitt.Key("foo")) {
		t.Error("key must remain present")
	}
}

func TestOffWithTypedNilWildcardClearsType(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)
	e.OnWildcardFunc(func(mitt.EventType, any) {})

	e.Off(mitt.Wildcard, (*mitt.WildcardHandler[any])(nil))

	assertHandlers(t, mustGet(t, m, mitt.Wildcard))
}

// Off must not corrupt a slice that a caller already holds.
func TestOffDoesNotAliasCallerSlice(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	a := e.OnFunc(mitt.Key("foo"), func(any) {})
	b := e.OnFunc(mitt.Key("foo"), func(any) {})
	c := e.OnFunc(mitt.Key("foo"), func(any) {})

	held := mustGet(t, m, mitt.Key("foo"))
	e.Off(mitt.Key("foo"), b)

	assertHandlers(t, held, a, b, c)
	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), a, c)
}

// Direct map mutation is part of the public contract: the upstream suite
// registers wildcard handlers via events.set('*', [star]).
func TestDirectMapMutationIsLive(t *testing.T) {
	e := mitt.New[any]()
	s := newSpy()

	e.All().Set(mitt.Wildcard, regs(mitt.NewWildcardHandler(s.wildcard())))
	e.Emit(mitt.Key("foo"), "x")
	if !s.calledOnce() {
		t.Fatalf("calls = %d, want 1", s.count())
	}

	e.All().Clear()
	e.Emit(mitt.Key("foo"), "x")
	if s.count() != 1 {
		t.Fatalf("calls after Clear = %d, want 1", s.count())
	}
	if e.All().Len() != 0 {
		t.Fatalf("Len after Clear = %d, want 0", e.All().Len())
	}
}

// Verified against the original: a handler that overwrites the live handler
// slice in place does not affect the emit already in flight, because dispatch
// runs over a copy. The mutation does persist for later emits.
func TestInPlaceSliceMutationDuringDispatchIsIgnored(t *testing.T) {
	e := mitt.New[any]()
	var log []string

	a := mitt.NewHandler[any](func(any) { log = append(log, "a") })
	b := mitt.NewHandler[any](func(any) { log = append(log, "b") })
	replacement := mitt.NewHandler[any](func(any) { log = append(log, "REPLACEMENT") })

	mutator := mitt.NewHandler[any](func(any) {
		live, _ := e.All().Get(mitt.Key("foo"))
		live[2] = replacement
	})

	e.On(mitt.Key("foo"), mutator)
	e.On(mitt.Key("foo"), a)
	e.On(mitt.Key("foo"), b)

	e.EmitVoid(mitt.Key("foo"))

	if len(log) != 2 || log[0] != "a" || log[1] != "b" {
		t.Fatalf("log = %v, want [a b]", log)
	}

	assertHandlers(t, mustGet(t, e.All(), mitt.Key("foo")), mutator, a, replacement)

	log = nil
	e.EmitVoid(mitt.Key("foo"))
	if len(log) != 2 || log[0] != "a" || log[1] != "REPLACEMENT" {
		t.Fatalf("second emit log = %v, want [a REPLACEMENT]", log)
	}
}
