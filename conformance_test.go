package mitt_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	mitt "github.com/developit/mitt-go"
)

// Differential test against the original library.
//
// testdata/conformance/programs.json is a list of op-scripts. expected.json
// holds the trace each script produces when run against mitt's unmodified
// src/index.ts under Node. This test replays the same scripts through the Go
// port and requires byte-identical traces, covering both the handler calls
// (with their exact arguments) and the state of `all` at each checkpoint.

type encVal struct {
	T string `json:"t"`
	V any    `json:"v,omitempty"`
}

type stateEntry struct {
	Type     encVal   `json:"type"`
	Handlers []string `json:"handlers"`
}

type traceEntry struct {
	H     string       `json:"h,omitempty"`
	Args  []encVal     `json:"args,omitempty"`
	State []stateEntry `json:"state,omitempty"`
}

type opSpec struct {
	Op      string  `json:"op"`
	Type    encVal  `json:"type"`
	H       *string `json:"h"`
	Kind    string  `json:"kind"`
	Payload *encVal `json:"payload"`
}

type program struct {
	Name string   `json:"name"`
	Ops  []opSpec `json:"ops"`
}

func TestConformanceWithOriginalJavaScript(t *testing.T) {
	dir := filepath.Join("testdata", "conformance")

	programs := decodeJSON[[]program](t, filepath.Join(dir, "programs.json"))
	expected := decodeJSON[map[string][]traceEntry](t, filepath.Join(dir, "expected.json"))

	if len(programs) == 0 {
		t.Fatal("no conformance programs")
	}

	for _, prog := range programs {
		t.Run(prog.Name, func(t *testing.T) {
			want, ok := expected[prog.Name]
			if !ok {
				t.Fatalf("no recorded JavaScript trace for %q", prog.Name)
			}

			got := runProgram(t, prog)

			if !reflect.DeepEqual(normalizeTrace(got), normalizeTrace(want)) {
				t.Fatalf("trace mismatch\n go: %s\n js: %s",
					mustJSON(normalizeTrace(got)), mustJSON(normalizeTrace(want)))
			}
		})
	}

	for name := range expected {
		if !containsProgram(programs, name) {
			t.Errorf("recorded trace %q has no matching program", name)
		}
	}
}

func runProgram(t *testing.T, prog program) []traceEntry {
	t.Helper()

	e := mitt.New[any]()
	trace := []traceEntry{}

	symbols := map[string]*mitt.Symbol{}
	symbolNames := map[*mitt.Symbol]string{}
	symbolFor := func(name string) *mitt.Symbol {
		if s, ok := symbols[name]; ok {
			return s
		}
		s := mitt.Sym(name)
		symbols[name] = s
		symbolNames[s] = name
		return s
	}

	var encode func(v any) encVal
	encode = func(v any) encVal {
		switch x := v.(type) {
		case nil:
			return encVal{T: "undef"}
		case mitt.Key:
			return encVal{T: "str", V: string(x)}
		case *mitt.Symbol:
			return encVal{T: "sym", V: symbolNames[x]}
		case *mitt.Raw:
			return encode(x.Value())
		case string:
			return encVal{T: "str", V: x}
		case int64:
			return encVal{T: "num", V: x}
		default:
			return encVal{T: "other", V: fmt.Sprint(x)}
		}
	}

	decodeType := func(v encVal) mitt.EventType {
		if v.T == "sym" {
			return symbolFor(v.V.(string))
		}
		return mitt.Key(v.V.(string))
	}
	decodeValue := func(v *encVal) any {
		switch {
		case v == nil, v.T == "undef":
			return nil
		case v.T == "sym":
			return symbolFor(v.V.(string))
		case v.T == "num":
			return toInt64(v.V)
		default:
			return v.V
		}
	}

	handlers := map[string]mitt.Registration[any]{}
	names := map[mitt.Registration[any]]string{}
	handlerFor := func(name, kind string) mitt.Registration[any] {
		if h, ok := handlers[name]; ok {
			return h
		}
		var h mitt.Registration[any]
		if kind == "wild" {
			h = mitt.NewWildcardHandler[any](func(tp mitt.EventType, ev any) {
				trace = append(trace, traceEntry{H: name, Args: []encVal{encode(tp), encode(ev)}})
			})
		} else {
			h = mitt.NewHandler[any](func(ev any) {
				trace = append(trace, traceEntry{H: name, Args: []encVal{encode(ev)}})
			})
		}
		handlers[name] = h
		names[h] = name
		return h
	}

	snapshot := func() []stateEntry {
		out := []stateEntry{}
		for _, k := range e.All().Keys() {
			hs, _ := e.All().Get(k)
			hn := []string{}
			for _, h := range hs {
				hn = append(hn, names[h])
			}
			out = append(out, stateEntry{Type: encode(k), Handlers: hn})
		}
		return out
	}

	for i, op := range prog.Ops {
		switch op.Op {
		case "on":
			e.On(decodeType(op.Type), handlerFor(*op.H, op.Kind))
		case "off":
			if op.H == nil {
				e.OffAll(decodeType(op.Type))
			} else {
				e.Off(decodeType(op.Type), handlerFor(*op.H, "single"))
			}
		case "emit":
			e.Emit(decodeType(op.Type), decodeValue(op.Payload))
		case "state":
			trace = append(trace, traceEntry{State: snapshot()})
		default:
			t.Fatalf("op %d: unknown op %q", i, op.Op)
		}
	}
	return trace
}

func decodeJSON[T any](t *testing.T, path string) T {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.UseNumber()

	var out T
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

// The JSON decoder yields json.Number; the Go run yields int64. Canonicalise
// both to int64 so the comparison is on values, not on their spelling.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			panic(err)
		}
		return i
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		panic(fmt.Sprintf("not a number: %#v", v))
	}
}

func normalizeTrace(entries []traceEntry) []traceEntry {
	out := make([]traceEntry, 0, len(entries))
	for _, entry := range entries {
		normalized := traceEntry{H: entry.H}
		if entry.Args != nil {
			normalized.Args = make([]encVal, len(entry.Args))
			for i, a := range entry.Args {
				normalized.Args[i] = normalizeVal(a)
			}
		}
		if entry.State != nil {
			normalized.State = make([]stateEntry, len(entry.State))
			for i, s := range entry.State {
				hs := s.Handlers
				if hs == nil {
					hs = []string{}
				}
				normalized.State[i] = stateEntry{Type: normalizeVal(s.Type), Handlers: hs}
			}
		}
		out = append(out, normalized)
	}
	return out
}

func normalizeVal(v encVal) encVal {
	if v.T == "num" {
		return encVal{T: "num", V: toInt64(v.V)}
	}
	return v
}

func containsProgram(programs []program, name string) bool {
	for _, p := range programs {
		if p.Name == name {
			return true
		}
	}
	return false
}

func mustJSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return buf.String()
}
