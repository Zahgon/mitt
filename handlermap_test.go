package mitt_test

import (
	"reflect"
	"strconv"
	"testing"

	mitt "github.com/developit/mitt-go"
)

// JavaScript's Map preserves insertion order and mitt's suite inspects `all`
// directly, so the Go port cannot use a plain (randomised) map.
func TestHandlerMapPreservesInsertionOrder(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	names := []string{"z", "a", "m", "b", "y"}
	for _, n := range names {
		e.OnFunc(mitt.Key(n), func(any) {})
	}

	want := make([]mitt.EventType, len(names))
	for i, n := range names {
		want[i] = mitt.Key(n)
	}
	if got := m.Keys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys = %v, want %v", got, want)
	}
}

func TestHandlerMapOrderAfterDeleteAndReinsert(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	for _, n := range []string{"a", "b", "c", "d"} {
		e.OnFunc(mitt.Key(n), func(any) {})
	}

	if !m.Delete(mitt.Key("b")) {
		t.Fatal("Delete(b) reported false")
	}
	if m.Delete(mitt.Key("b")) {
		t.Fatal("second Delete(b) must report false")
	}

	e.OnFunc(mitt.Key("b"), func(any) {})

	want := []mitt.EventType{mitt.Key("a"), mitt.Key("c"), mitt.Key("d"), mitt.Key("b")}
	if got := m.Keys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys = %v, want %v", got, want)
	}

	// Re-setting an existing key keeps its position.
	m.Set(mitt.Key("c"), regs())
	if got := m.Keys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys after Set = %v, want %v", got, want)
	}
}

func TestHandlerMapDeleteRemapsRemainingIndices(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	hs := make([]mitt.Registration[any], 0, 6)
	for i := 0; i < 6; i++ {
		h := mitt.NewHandler[any](func(any) {})
		hs = append(hs, h)
		m.Set(mitt.Key(strconv.Itoa(i)), regs(h))
	}

	m.Delete(mitt.Key("0"))
	m.Delete(mitt.Key("3"))

	for _, i := range []int{1, 2, 4, 5} {
		got, ok := m.Get(mitt.Key(strconv.Itoa(i)))
		if !ok {
			t.Fatalf("key %d missing after deletes", i)
		}
		assertHandlers(t, got, hs[i])
	}
	if m.Len() != 4 {
		t.Fatalf("Len = %d, want 4", m.Len())
	}
}

func TestHandlerMapRange(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	for _, n := range []string{"a", "b", "c"} {
		m.Set(mitt.Key(n), regs(mitt.NewHandler[any](func(any) {})))
	}

	var seen []string
	m.Range(func(k mitt.EventType, _ []mitt.Registration[any]) bool {
		seen = append(seen, string(k.(mitt.Key)))
		return true
	})
	if !reflect.DeepEqual(seen, []string{"a", "b", "c"}) {
		t.Fatalf("Range order = %v, want [a b c]", seen)
	}

	seen = nil
	m.Range(func(k mitt.EventType, _ []mitt.Registration[any]) bool {
		seen = append(seen, string(k.(mitt.Key)))
		return len(seen) < 2
	})
	if !reflect.DeepEqual(seen, []string{"a", "b"}) {
		t.Fatalf("early-stop Range = %v, want [a b]", seen)
	}
}

func TestHandlerMapRangeToleratesMutation(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	for _, n := range []string{"a", "b"} {
		m.Set(mitt.Key(n), regs())
	}

	count := 0
	m.Range(func(k mitt.EventType, _ []mitt.Registration[any]) bool {
		count++
		m.Delete(k)
		return true
	})
	if count != 2 {
		t.Fatalf("visited %d entries, want 2", count)
	}
	if m.Len() != 0 {
		t.Fatalf("Len = %d, want 0", m.Len())
	}
}

func TestHandlerMapGetAbsent(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	if hs, ok := m.Get(mitt.Key("nope")); ok || hs != nil {
		t.Fatalf("Get(absent) = %v, %v; want nil, false", hs, ok)
	}
	if m.Has(mitt.Key("nope")) {
		t.Error("Has(absent) = true")
	}
}

func TestKeyString(t *testing.T) {
	if got := mitt.Key("foo").String(); got != "foo" {
		t.Fatalf("Key.String = %q, want %q", got, "foo")
	}
	if mitt.Wildcard.String() != "*" {
		t.Fatalf("Wildcard = %q, want *", mitt.Wildcard.String())
	}
}

func TestRawAccessors(t *testing.T) {
	e := mitt.New[any]()
	var boxed *mitt.Raw
	e.OnWildcardFunc(func(t mitt.EventType, _ any) {
		if r, ok := t.(*mitt.Raw); ok {
			boxed = r
		}
	})
	e.Emit(mitt.Wildcard, 42)

	if boxed == nil {
		t.Fatal("expected a *Raw in the type position")
	}
	if boxed.Value() != 42 {
		t.Fatalf("Value = %v, want 42", boxed.Value())
	}
	if boxed.String() != "42" {
		t.Fatalf("String = %q, want the payload rendered transparently", boxed.String())
	}
}

func TestNilReceiverStringers(t *testing.T) {
	var s *mitt.Symbol
	if s.String() != "Symbol(<nil>)" {
		t.Errorf("nil Symbol.String = %q", s.String())
	}
	var r *mitt.Raw
	if r.String() != "<nil>" {
		t.Errorf("nil Raw.String = %q", r.String())
	}
	if r.Value() != nil {
		t.Errorf("nil Raw.Value = %v", r.Value())
	}
}

func TestHandlerCallDirect(t *testing.T) {
	var got any
	h := mitt.NewHandler[any](func(ev any) { got = ev })
	h.Call("x")
	if got != "x" {
		t.Fatalf("got = %v, want x", got)
	}

	var gotType mitt.EventType
	w := mitt.NewWildcardHandler[any](func(t mitt.EventType, ev any) { gotType, got = t, ev })
	w.Call(mitt.Key("k"), "y")
	if gotType != mitt.EventType(mitt.Key("k")) || got != "y" {
		t.Fatalf("got = %v, %v; want k, y", gotType, got)
	}
}
