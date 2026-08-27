package mitt_test

import (
	"testing"

	mitt "github.com/developit/mitt-go"
)

// The remaining tests exercise the JS arity-coercion paths that only trigger
// when the wildcard key is used as an ordinary event type.

func TestZeroValueHandlerMapIsAdopted(t *testing.T) {
	var m mitt.HandlerMap[any]
	e := mitt.NewWith(&m)

	s := newSpy()
	e.OnFunc(mitt.Key("foo"), s.handler())
	e.Emit(mitt.Key("foo"), 1)

	if !s.calledOnce() {
		t.Fatalf("count = %d, want 1", s.count())
	}
	if m.Len() != 1 {
		t.Fatalf("Len = %d, want 1", m.Len())
	}
}

func TestNilHandlerUnderWildcardPanicsOnWildcardDispatch(t *testing.T) {
	e := mitt.New[any]()
	e.On(mitt.Wildcard, mitt.NewHandler[any](nil))

	defer func() {
		if recover() == nil {
			t.Error("expected panic from nil handler on wildcard dispatch")
		}
	}()
	e.Emit(mitt.Key("foo"), 1)
}

// A payload that is itself an EventType is passed straight through into the
// type position, no boxing required.
func TestEventTypePayloadIsNotBoxed(t *testing.T) {
	e := mitt.New[any]()
	var got []mitt.EventType
	e.OnWildcardFunc(func(tp mitt.EventType, _ any) { got = append(got, tp) })

	e.Emit(mitt.Wildcard, mitt.Key("inner"))

	if len(got) != 2 {
		t.Fatalf("calls = %d, want 2", len(got))
	}
	if got[0] != mitt.EventType(mitt.Key("inner")) {
		t.Fatalf("got[0] = %#v, want Key(inner) unboxed", got[0])
	}
}

// A plain string payload becomes a Key rather than a Raw box.
func TestStringPayloadBecomesKey(t *testing.T) {
	e := mitt.New[string]()
	var got []mitt.EventType
	e.OnWildcard(mitt.NewWildcardHandler[string](func(tp mitt.EventType, _ string) {
		got = append(got, tp)
	}))

	e.Emit(mitt.Wildcard, "abc")

	if len(got) != 2 {
		t.Fatalf("calls = %d, want 2", len(got))
	}
	if got[0] != mitt.EventType(mitt.Key("abc")) {
		t.Fatalf("got[0] = %#v, want Key(abc)", got[0])
	}
}

// A boxed Raw round-trips: re-emitting it as the event type unwraps back to the
// original payload for a single-argument handler.
func TestRawUnboxesForSingleArgHandler(t *testing.T) {
	probe := mitt.New[any]()
	var seen []mitt.EventType
	probe.OnWildcardFunc(func(tp mitt.EventType, _ any) { seen = append(seen, tp) })
	probe.Emit(mitt.Wildcard, 99)

	// Emitting the wildcard key invokes the handler twice; the boxed payload
	// arrives on the first call, via the type lookup.
	boxed := seen[0]
	if _, ok := boxed.(*mitt.Raw); !ok {
		t.Fatalf("expected *Raw, got %#v", boxed)
	}

	e := mitt.New[int]()
	var got []int
	e.On(mitt.Wildcard, mitt.NewHandler[int](func(n int) { got = append(got, n) }))

	e.Emit(boxed, 7)

	if len(got) != 1 || got[0] != 99 {
		t.Fatalf("got = %v, want [99] unboxed from Raw", got)
	}
}

// When the event type cannot be represented in the payload type at all, the
// handler receives the zero value rather than panicking, matching JS passing a
// value the handler simply ignores.
func TestUnconvertibleTypeYieldsZeroPayload(t *testing.T) {
	e := mitt.New[int]()
	var got []int
	e.On(mitt.Wildcard, mitt.NewHandler[int](func(n int) { got = append(got, n) }))

	e.Emit(mitt.Sym("s"), 5)

	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("got = %v, want [0]", got)
	}
}
