package mitt_test

import (
	"testing"

	mitt "github.com/developit/mitt-go"
)

func BenchmarkEmitSingleHandler(b *testing.B) {
	e := mitt.New[int]()
	sink := 0
	e.OnFunc(mitt.Key("foo"), func(n int) { sink += n })

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Emit(mitt.Key("foo"), 1)
	}
	_ = sink
}

func BenchmarkEmitTenHandlers(b *testing.B) {
	e := mitt.New[int]()
	sink := 0
	for j := 0; j < 10; j++ {
		e.OnFunc(mitt.Key("foo"), func(n int) { sink += n })
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Emit(mitt.Key("foo"), 1)
	}
	_ = sink
}

func BenchmarkEmitWithWildcard(b *testing.B) {
	e := mitt.New[int]()
	sink := 0
	e.OnFunc(mitt.Key("foo"), func(n int) { sink += n })
	e.OnWildcardFunc(func(_ mitt.EventType, n int) { sink += n })

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Emit(mitt.Key("foo"), 1)
	}
	_ = sink
}

func BenchmarkEmitNoHandlers(b *testing.B) {
	e := mitt.New[int]()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Emit(mitt.Key("foo"), 1)
	}
}

func BenchmarkOnOff(b *testing.B) {
	e := mitt.New[int]()
	fn := func(int) {}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h := e.OnFunc(mitt.Key("foo"), fn)
		e.Off(mitt.Key("foo"), h)
	}
}

func BenchmarkSyncEmitParallel(b *testing.B) {
	e := mitt.NewSync[int]()
	e.OnFunc(mitt.Key("foo"), func(int) {})

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			e.Emit(mitt.Key("foo"), 1)
		}
	})
}
