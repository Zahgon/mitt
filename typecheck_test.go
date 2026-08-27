package mitt_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	mitt "github.com/developit/mitt-go"
)

// Port of test/test-types-compilation.ts.
//
// The positive half lives in this file: if it compiles, the API accepts the
// same shapes the TS original accepts. The negative half lives in testdata/fail
// and is checked by TestRejectedProgramsDoNotCompile, which is the Go analogue
// of `// @ts-expect-error`.

func TestAcceptedProgramsCompile(t *testing.T) {
	type someEvent struct{ Name string }

	emitter := mitt.New[any]()

	fooHandler := func(x string) { _ = x }
	barHandler := func(x int) { _ = x }
	wildcardHandler := func(_ mitt.EventType, _ any) {}

	// on
	mitt.OnTyped[any, string](emitter, mitt.Key("foo"), fooHandler)
	mitt.OnTyped[any, int](emitter, mitt.Key("bar"), barHandler)
	emitter.OnWildcardFunc(wildcardHandler)
	// A one-argument handler on '*' is legal; it receives the event key.
	emitter.On(mitt.Wildcard, mitt.NewHandler[any](func(any) {}))

	// off
	h := mitt.OnTyped[any, string](emitter, mitt.Key("foo"), fooHandler)
	emitter.Off(mitt.Key("foo"), h)
	w := emitter.OnWildcardFunc(wildcardHandler)
	emitter.Off(mitt.Wildcard, w)
	emitter.OffAll(mitt.Key("foo"))

	// emit
	mitt.EmitTyped[any, someEvent](emitter, mitt.Key("someEvent"), someEvent{Name: "jack"})
	mitt.EmitTyped[any, string](emitter, mitt.Key("foo"), "string")
	emitter.EmitVoid(mitt.Key("bar"))
	emitter.Emit(mitt.Key("bar"), 1)

	// symbols and the exposed map
	sym := mitt.Sym("evt")
	emitter.OnFunc(sym, func(any) {})
	var _ *mitt.HandlerMap[any] = emitter.All()

	// a strictly typed emitter
	strict := mitt.New[string]()
	sh := strict.OnFunc(mitt.Key("foo"), func(string) {})
	strict.Emit(mitt.Key("foo"), "ok")
	strict.Off(mitt.Key("foo"), sh)
}

func TestRejectedProgramsDoNotCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile-failure checks in -short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	entries, err := filepath.Glob(filepath.Join("testdata", "fail", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no compile-failure fixtures found")
	}

	for _, entry := range entries {
		t.Run(filepath.Base(entry), func(t *testing.T) {
			cmd := exec.Command("go", "build", "-o", os.DevNull, entry)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("%s compiled successfully but must be rejected", entry)
			}
			t.Logf("rejected as expected:\n%s", out)
		})
	}
}
