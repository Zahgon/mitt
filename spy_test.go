package mitt_test

import (
	"reflect"
	"sync"
	"testing"

	mitt "github.com/developit/mitt-go"
)

type call struct {
	Type  mitt.EventType
	Event any
}

type spy struct {
	mu    sync.Mutex
	calls []call
}

func newSpy() *spy { return &spy{} }

func (s *spy) handler() func(any) {
	return func(ev any) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.calls = append(s.calls, call{Event: ev})
	}
}

func (s *spy) wildcard() func(mitt.EventType, any) {
	return func(t mitt.EventType, ev any) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.calls = append(s.calls, call{Type: t, Event: ev})
	}
}

func (s *spy) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *spy) calledOnce() bool { return s.count() == 1 }

func (s *spy) resetHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = nil
}

func (s *spy) calledWith(idx int, want call) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx >= len(s.calls) {
		return false
	}
	return reflect.DeepEqual(s.calls[idx], want)
}

func regs(hs ...mitt.Registration[any]) []mitt.Registration[any] { return hs }

func mustGet(t *testing.T, m *mitt.HandlerMap[any], k mitt.EventType) []mitt.Registration[any] {
	t.Helper()
	hs, ok := m.Get(k)
	if !ok {
		t.Fatalf("expected key %v to be present", k)
	}
	return hs
}

func assertHandlers(t *testing.T, got []mitt.Registration[any], want ...mitt.Registration[any]) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("handler count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("handler[%d] identity mismatch", i)
		}
	}
}
