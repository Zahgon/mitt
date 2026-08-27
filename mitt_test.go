package mitt_test

import (
	"testing"

	mitt "github.com/developit/mitt-go"
)

// Port of test/index_test.ts, describe('mitt').

func TestDefaultExportShouldBeAFunction(t *testing.T) {
	e := mitt.New[any]()
	if e == nil {
		t.Fatal("New returned nil")
	}
	if e.All() == nil {
		t.Fatal("All() returned nil")
	}
}

func TestShouldAcceptAnOptionalEventHandlerMap(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	a, b := newSpy(), newSpy()
	m.Set(mitt.Key("foo"), regs(mitt.NewHandler(a.handler()), mitt.NewHandler(b.handler())))

	e := mitt.NewWith(m)
	e.EmitVoid(mitt.Key("foo"))

	if !a.calledOnce() {
		t.Errorf("a called %d times, want 1", a.count())
	}
	if !b.calledOnce() {
		t.Errorf("b called %d times, want 1", b.count())
	}
}

func TestNewWithNilMap(t *testing.T) {
	e := mitt.NewWith[any](nil)
	if e.All() == nil {
		t.Fatal("expected a fresh handler map")
	}
	e.OnFunc(mitt.Key("foo"), func(any) {})
	if e.All().Len() != 1 {
		t.Fatalf("Len = %d, want 1", e.All().Len())
	}
}

// Port of describe('mitt#') -> describe('properties').

func TestShouldExposeTheEventHandlerMap(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)
	if e.All() != m {
		t.Fatal("All() must return the same map that was passed in")
	}
}

// Port of describe('on()').

func TestOnShouldBeAFunction(t *testing.T) {
	var on func(mitt.EventType, mitt.Registration[any]) = mitt.New[any]().On
	if on == nil {
		t.Fatal("On is not callable")
	}
}

func TestOnShouldRegisterHandlerForNewType(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	foo := e.OnFunc(mitt.Key("foo"), func(any) {})

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), foo)
}

func TestOnShouldRegisterHandlersForAnyTypeStrings(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	// 'constructor' is the JS prototype-pollution canary from the original suite.
	for _, name := range []string{"constructor", "__proto__", "", "has spaces", "日本語", "toString"} {
		h := e.OnFunc(mitt.Key(name), func(any) {})
		assertHandlers(t, mustGet(t, m, mitt.Key(name)), h)
	}
}

func TestOnShouldAppendHandlerForExistingType(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	foo := e.OnFunc(mitt.Key("foo"), func(any) {})
	bar := e.OnFunc(mitt.Key("foo"), func(any) {})

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), foo, bar)
}

func TestOnShouldNotNormalizeCase(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	foo := mitt.NewHandler[any](func(any) {})
	e.On(mitt.Key("FOO"), foo)
	e.On(mitt.Key("Bar"), foo)
	e.On(mitt.Key("baz:baT!"), foo)

	assertHandlers(t, mustGet(t, m, mitt.Key("FOO")), foo)
	if m.Has(mitt.Key("foo")) {
		t.Error(`"foo" must not exist after registering "FOO"`)
	}
	assertHandlers(t, mustGet(t, m, mitt.Key("Bar")), foo)
	if m.Has(mitt.Key("bar")) {
		t.Error(`"bar" must not exist after registering "Bar"`)
	}
	assertHandlers(t, mustGet(t, m, mitt.Key("baz:baT!")), foo)
	if m.Has(mitt.Key("baz:bat!")) {
		t.Error(`"baz:bat!" must not exist after registering "baz:baT!"`)
	}
}

func TestOnCanTakeSymbolsForEventTypes(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	eventType := mitt.Sym("eventType")
	foo := e.OnFunc(eventType, func(any) {})

	assertHandlers(t, mustGet(t, m, eventType), foo)
}

func TestSymbolIdentityIsUnique(t *testing.T) {
	a, b := mitt.Sym("evt"), mitt.Sym("evt")
	if a == b {
		t.Fatal("two symbols with the same description must not be equal")
	}
	if a.Description() != "evt" {
		t.Errorf("Description = %q, want %q", a.Description(), "evt")
	}
	if a.String() != "Symbol(evt)" {
		t.Errorf("String = %q, want %q", a.String(), "Symbol(evt)")
	}

	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)
	sa, sb := newSpy(), newSpy()
	e.OnFunc(a, sa.handler())
	e.OnFunc(b, sb.handler())

	e.Emit(a, "payload")
	if !sa.calledOnce() || sb.count() != 0 {
		t.Fatalf("symbol keys collided: a=%d b=%d", sa.count(), sb.count())
	}
}

func TestOnShouldAddDuplicateListeners(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	foo := mitt.NewHandler[any](func(any) {})
	e.On(mitt.Key("foo"), foo)
	e.On(mitt.Key("foo"), foo)

	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), foo, foo)
}

func TestDuplicateHandlerFiresTwice(t *testing.T) {
	e := mitt.New[any]()
	s := newSpy()
	h := mitt.NewHandler(s.handler())
	e.On(mitt.Key("foo"), h)
	e.On(mitt.Key("foo"), h)

	e.Emit(mitt.Key("foo"), 1)

	if s.count() != 2 {
		t.Fatalf("count = %d, want 2", s.count())
	}
}

// Port of describe('off()').

func TestOffShouldBeAFunction(t *testing.T) {
	var off func(mitt.EventType, mitt.Registration[any]) = mitt.New[any]().Off
	if off == nil {
		t.Fatal("Off is not callable")
	}
}

func TestOffShouldRemoveHandlerForType(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	foo := e.OnFunc(mitt.Key("foo"), func(any) {})
	e.Off(mitt.Key("foo"), foo)

	if hs := mustGet(t, m, mitt.Key("foo")); len(hs) != 0 {
		t.Fatalf("len = %d, want 0", len(hs))
	}
}

func TestOffShouldNotNormalizeCase(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	foo := mitt.NewHandler[any](func(any) {})
	e.On(mitt.Key("FOO"), foo)
	e.On(mitt.Key("Bar"), foo)
	e.On(mitt.Key("baz:bat!"), foo)

	e.Off(mitt.Key("FOO"), foo)
	e.Off(mitt.Key("Bar"), foo)
	e.Off(mitt.Key("baz:baT!"), foo) // different case: must not touch "baz:bat!"

	if hs := mustGet(t, m, mitt.Key("FOO")); len(hs) != 0 {
		t.Errorf(`"FOO" len = %d, want 0`, len(hs))
	}
	if m.Has(mitt.Key("foo")) {
		t.Error(`"foo" must not exist`)
	}
	if hs := mustGet(t, m, mitt.Key("Bar")); len(hs) != 0 {
		t.Errorf(`"Bar" len = %d, want 0`, len(hs))
	}
	if m.Has(mitt.Key("bar")) {
		t.Error(`"bar" must not exist`)
	}
	if hs := mustGet(t, m, mitt.Key("baz:bat!")); len(hs) != 1 {
		t.Errorf(`"baz:bat!" len = %d, want 1`, len(hs))
	}
	if m.Has(mitt.Key("baz:baT!")) {
		t.Error(`off must not create an entry for an unregistered type`)
	}
}

func TestOffShouldRemoveOnlyTheFirstMatchingListener(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	foo := mitt.NewHandler[any](func(any) {})
	e.On(mitt.Key("foo"), foo)
	e.On(mitt.Key("foo"), foo)

	e.Off(mitt.Key("foo"), foo)
	assertHandlers(t, mustGet(t, m, mitt.Key("foo")), foo)

	e.Off(mitt.Key("foo"), foo)
	assertHandlers(t, mustGet(t, m, mitt.Key("foo")))
}

func TestOffTypeShouldRemoveAllHandlersOfTheGivenType(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	e.OnFunc(mitt.Key("foo"), func(any) {})
	e.OnFunc(mitt.Key("foo"), func(any) {})
	e.OnFunc(mitt.Key("bar"), func(any) {})

	e.OffAll(mitt.Key("foo"))
	assertHandlers(t, mustGet(t, m, mitt.Key("foo")))
	if !m.Has(mitt.Key("foo")) {
		t.Error("off(type) must leave the key present with an empty list, not delete it")
	}
	if hs := mustGet(t, m, mitt.Key("bar")); len(hs) != 1 {
		t.Errorf(`"bar" len = %d, want 1`, len(hs))
	}

	e.OffAll(mitt.Key("bar"))
	assertHandlers(t, mustGet(t, m, mitt.Key("bar")))
}

// Port of describe('emit()').

func TestEmitShouldBeAFunction(t *testing.T) {
	var emit func(mitt.EventType, any) = mitt.New[any]().Emit
	if emit == nil {
		t.Fatal("Emit is not callable")
	}
}

func TestEmitShouldInvokeHandlerForType(t *testing.T) {
	e := mitt.New[any]()
	event := map[string]string{"a": "b"}
	got := make([]any, 0, 1)

	e.OnFunc(mitt.Key("foo"), func(ev any) { got = append(got, ev) })
	e.Emit(mitt.Key("foo"), event)

	if len(got) != 1 {
		t.Fatalf("handler called %d times, want 1", len(got))
	}
	m, ok := got[0].(map[string]string)
	if !ok || m["a"] != "b" {
		t.Fatalf("payload = %#v, want %#v", got[0], event)
	}
}

func TestEmitShouldNotIgnoreCase(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	onFoo, onFOO := newSpy(), newSpy()
	m.Set(mitt.Key("Foo"), regs(mitt.NewHandler(onFoo.handler())))
	m.Set(mitt.Key("FOO"), regs(mitt.NewHandler(onFOO.handler())))

	e.Emit(mitt.Key("Foo"), "Foo arg")
	e.Emit(mitt.Key("FOO"), "FOO arg")

	if !onFoo.calledOnce() || !onFoo.calledWith(0, call{Event: "Foo arg"}) {
		t.Errorf("onFoo calls = %d", onFoo.count())
	}
	if !onFOO.calledOnce() || !onFOO.calledWith(0, call{Event: "FOO arg"}) {
		t.Errorf("onFOO calls = %d", onFOO.count())
	}
}

func TestShouldInvokeWildcardHandlers(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	star := newSpy()
	ea := map[string]string{"a": "a"}
	eb := map[string]string{"b": "b"}

	m.Set(mitt.Wildcard, regs(mitt.NewWildcardHandler(star.wildcard())))

	e.Emit(mitt.Key("foo"), ea)
	if !star.calledOnce() || !star.calledWith(0, call{Type: mitt.Key("foo"), Event: ea}) {
		t.Fatalf("star not called with (foo, ea); calls = %d", star.count())
	}
	star.resetHistory()

	e.Emit(mitt.Key("bar"), eb)
	if !star.calledOnce() || !star.calledWith(0, call{Type: mitt.Key("bar"), Event: eb}) {
		t.Fatalf("star not called with (bar, eb); calls = %d", star.count())
	}
}

func TestEmitUnknownTypeIsNoop(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	e := mitt.NewWith(m)

	e.Emit(mitt.Key("nobody-listening"), 1)

	if m.Has(mitt.Key("nobody-listening")) {
		t.Error("emit must not create a handler-map entry")
	}
	if m.Len() != 0 {
		t.Errorf("Len = %d, want 0", m.Len())
	}
}
