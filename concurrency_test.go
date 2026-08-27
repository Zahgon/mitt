package mitt_test

import (
	"sync"
	"sync/atomic"
	"testing"

	mitt "github.com/developit/mitt-go"
)

func TestSyncEmitterConcurrentOnOffEmit(t *testing.T) {
	e := mitt.NewSync[any]()
	var fired atomic.Int64

	const workers = 16
	const iterations = 200

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			key := mitt.Key("evt")
			for j := 0; j < iterations; j++ {
				h := e.OnFunc(key, func(any) { fired.Add(1) })
				e.Emit(key, w)
				e.Off(key, h)
			}
		}(w)
	}
	wg.Wait()

	if fired.Load() == 0 {
		t.Fatal("expected handlers to fire")
	}
	if hs, ok := e.All().Get(mitt.Key("evt")); ok && len(hs) != 0 {
		t.Fatalf("leftover handlers = %d, want 0", len(hs))
	}
}

func TestSyncEmitterConcurrentMapAccess(t *testing.T) {
	e := mitt.NewSync[any]()
	m := e.All()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			k := mitt.Key("k")
			for j := 0; j < 100; j++ {
				m.Set(k, regs(mitt.NewHandler[any](func(any) {})))
				m.Has(k)
				m.Get(k)
				m.Keys()
				m.Len()
				m.Range(func(mitt.EventType, []mitt.Registration[any]) bool { return true })
				if i%2 == 0 {
					m.Delete(k)
				}
			}
		}(i)
	}
	wg.Wait()
}

// Snapshot-then-dispatch releases the lock before invoking handlers, so a
// handler may re-enter the emitter without deadlocking.
func TestSyncEmitterHandlerReentrancyDoesNotDeadlock(t *testing.T) {
	e := mitt.NewSync[any]()
	done := make(chan struct{})

	e.OnFunc(mitt.Key("outer"), func(any) {
		e.OnFunc(mitt.Key("inner"), func(any) { close(done) })
		e.Emit(mitt.Key("inner"), nil)
		e.OffAll(mitt.Key("inner"))
	})

	e.Emit(mitt.Key("outer"), nil)

	select {
	case <-done:
	default:
		t.Fatal("inner handler never ran")
	}
}

func TestSyncEmitterSemanticsMatchPlain(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func() *mitt.Emitter[any]
	}{
		{"plain", mitt.New[any]},
		{"sync", mitt.NewSync[any]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.make()
			var order []string

			h := mitt.NewHandler[any](func(any) { order = append(order, "typed") })
			e.On(mitt.Key("foo"), h)
			e.On(mitt.Key("foo"), h)
			e.OnWildcardFunc(func(mitt.EventType, any) { order = append(order, "wild") })

			e.Emit(mitt.Key("foo"), nil)
			e.Off(mitt.Key("foo"), h)
			e.Emit(mitt.Key("foo"), nil)

			want := []string{"typed", "typed", "wild", "typed", "wild"}
			if len(order) != len(want) {
				t.Fatalf("order = %v, want %v", order, want)
			}
			for i := range want {
				if order[i] != want[i] {
					t.Fatalf("order = %v, want %v", order, want)
				}
			}
		})
	}
}

func TestNewSyncWith(t *testing.T) {
	m := mitt.NewHandlerMap[any]()
	s := newSpy()
	m.Set(mitt.Key("foo"), regs(mitt.NewHandler(s.handler())))

	e := mitt.NewSyncWith(m)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				e.Emit(mitt.Key("foo"), 1)
			}
		}()
	}
	wg.Wait()

	if s.count() != 400 {
		t.Fatalf("count = %d, want 400", s.count())
	}
}

func TestNewSyncWithNil(t *testing.T) {
	e := mitt.NewSyncWith[any](nil)
	if e.All() == nil {
		t.Fatal("expected a fresh handler map")
	}
	e.OnFunc(mitt.Key("x"), func(any) {})
	if e.All().Len() != 1 {
		t.Fatalf("Len = %d, want 1", e.All().Len())
	}
}
