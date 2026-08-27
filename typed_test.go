package mitt_test

import (
	"testing"

	mitt "github.com/developit/mitt-go"
)

type someEventData struct {
	Name string
}

func TestOnTypedReceivesAssertedPayload(t *testing.T) {
	e := mitt.New[any]()
	var got []string

	mitt.OnTyped[any, string](e, mitt.Key("foo"), func(s string) { got = append(got, s) })
	mitt.EmitTyped[any, string](e, mitt.Key("foo"), "hello")

	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("got = %v, want [hello]", got)
	}
}

func TestOnTypedStructPayload(t *testing.T) {
	e := mitt.New[any]()
	var got []someEventData

	mitt.OnTyped[any, someEventData](e, mitt.Key("someEvent"), func(d someEventData) {
		got = append(got, d)
	})
	mitt.EmitTyped[any, someEventData](e, mitt.Key("someEvent"), someEventData{Name: "jack"})

	if len(got) != 1 || got[0].Name != "jack" {
		t.Fatalf("got = %v, want [{jack}]", got)
	}
}

func TestOnTypedPanicsOnMismatch(t *testing.T) {
	e := mitt.New[any]()
	mitt.OnTyped[any, string](e, mitt.Key("foo"), func(string) {})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on payload type mismatch")
		}
	}()
	e.Emit(mitt.Key("foo"), 42)
}

// A missing payload is JS `undefined`, not a type error: OnTyped delivers the
// zero value rather than panicking, matching `emit(type)` on an optional event.
func TestOnTypedVoidEmitDeliversZero(t *testing.T) {
	e := mitt.New[any]()
	var got []int
	mitt.OnTyped[any, int](e, mitt.Key("bar"), func(n int) { got = append(got, n) })

	e.EmitVoid(mitt.Key("bar"))

	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("got = %v, want [0]", got)
	}
}

func TestEmitTypedPanicsWhenPayloadNotAssignable(t *testing.T) {
	e := mitt.New[string]()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic emitting an int on Emitter[string]")
		}
	}()
	mitt.EmitTyped[string, int](e, mitt.Key("foo"), 1)
}

func TestOnWildcardTyped(t *testing.T) {
	e := mitt.New[any]()
	type rec struct {
		t mitt.EventType
		s string
	}
	var got []rec

	mitt.OnWildcardTyped[any, string](e, func(tp mitt.EventType, s string) {
		got = append(got, rec{tp, s})
	})

	mitt.EmitTyped[any, string](e, mitt.Key("foo"), "a")
	mitt.EmitTyped[any, string](e, mitt.Key("bar"), "b")

	if len(got) != 2 {
		t.Fatalf("calls = %d, want 2", len(got))
	}
	if got[0].t != mitt.EventType(mitt.Key("foo")) || got[0].s != "a" {
		t.Errorf("call 0 = %v", got[0])
	}
	if got[1].t != mitt.EventType(mitt.Key("bar")) || got[1].s != "b" {
		t.Errorf("call 1 = %v", got[1])
	}
}

func TestOnTypedHandleCanBeRemoved(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	h := mitt.OnTyped[any, string](e, mitt.Key("foo"), func(string) {})
	e.Off(mitt.Key("foo"), h)

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")))
}

// Distinct closures produced from the same function literal must remain
// distinct registrations. reflect.ValueOf(fn).Pointer() collapses them onto one
// code pointer, which is precisely why handlers are pointer handles.
func TestClosuresFromSameLiteralAreDistinctRegistrations(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	makeHandler := func(tag string, sink *[]string) *mitt.Handler[any] {
		return mitt.NewHandler[any](func(any) { *sink = append(*sink, tag) })
	}

	var sink []string
	a := makeHandler("a", &sink)
	b := makeHandler("b", &sink)
	e.On(mitt.Key("foo"), a)
	e.On(mitt.Key("foo"), b)

	e.Off(mitt.Key("foo"), a)

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), b)

	e.Emit(mitt.Key("foo"), nil)
	if len(sink) != 1 || sink[0] != "b" {
		t.Fatalf("sink = %v, want [b]", sink)
	}
}
