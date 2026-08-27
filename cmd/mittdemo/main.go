// Command mittdemo exercises the emitter through a handful of deterministic
// scenarios. Its output is byte-for-byte identical to demo/cli.ts in the
// TypeScript original, which is what the behavioural parity fixtures compare.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	mitt "github.com/developit/mitt-go"
)

// undefinedLabel mirrors what JavaScript's String() prints for a missing
// payload, so a bare emit reads as "undefined" and not as Go's "<nil>".
const undefinedLabel = "undefined"

const usageFormat = "usage: mittdemo <%s>\n"

// trace collects the lines a scenario emits so tests can assert them directly
// instead of capturing process stdout.
type trace struct {
	lines []string
}

func (t *trace) log(format string, args ...any) {
	t.lines = append(t.lines, fmt.Sprintf(format, args...))
}

func (t *trace) String() string {
	return strings.Join(t.lines, "\n")
}

func show(v any) string {
	if v == nil {
		return undefinedLabel
	}
	return fmt.Sprint(v)
}

func count(e *mitt.Emitter[any], t mitt.EventType) int {
	hs, _ := e.All().Get(t)
	return len(hs)
}

func basic(out *trace) {
	events := mitt.New[any]()
	handler := mitt.NewHandler(func(evt any) { out.log("foo=%s", show(evt)) })
	events.On(mitt.Key("foo"), handler)
	events.Emit(mitt.Key("foo"), 1)
	events.Off(mitt.Key("foo"), handler)
	events.Emit(mitt.Key("foo"), 2)
	out.log("remaining=%d", count(events, mitt.Key("foo")))
}

func wildcard(out *trace) {
	events := mitt.New[any]()
	events.On(mitt.Wildcard, mitt.NewWildcardHandler(func(t mitt.EventType, evt any) {
		out.log("*:%s=%s", t, show(evt))
	}))
	events.On(mitt.Key("a"), mitt.NewHandler(func(evt any) { out.log("a=%s", show(evt)) }))
	events.Emit(mitt.Key("a"), "x")
	events.Emit(mitt.Key("b"), "y")
}

func duplicate(out *trace) {
	events := mitt.New[any]()
	handler := mitt.NewHandler(func(evt any) { out.log("h=%s", show(evt)) })
	events.On(mitt.Key("d"), handler)
	events.On(mitt.Key("d"), handler)
	events.Emit(mitt.Key("d"), 1)
	events.Off(mitt.Key("d"), handler)
	out.log("left=%d", count(events, mitt.Key("d")))
	events.Emit(mitt.Key("d"), 2)
}

func offUnregistered(out *trace) {
	events := mitt.New[any]()
	a := mitt.NewHandler(func(evt any) { out.log("a=%s", show(evt)) })
	b := mitt.NewHandler(func(evt any) { out.log("b=%s", show(evt)) })
	events.On(mitt.Key("u"), a)
	events.Off(mitt.Key("u"), b)
	out.log("len=%d", count(events, mitt.Key("u")))
	events.Emit(mitt.Key("u"), "z")
}

func emitStar(out *trace) {
	events := mitt.New[any]()
	events.On(mitt.Wildcard, mitt.NewWildcardHandler(func(t mitt.EventType, evt any) {
		out.log("w:%s|%s", t, show(evt))
	}))
	events.Emit(mitt.Wildcard, "p")
}

func symbols(out *trace) {
	events := mitt.New[any]()
	key := mitt.Sym("s")
	events.On(key, mitt.NewHandler(func(evt any) { out.log("s=%s", show(evt)) }))
	events.On(mitt.Wildcard, mitt.NewWildcardHandler(func(t mitt.EventType, evt any) {
		out.log("*:%s=%s", t, show(evt))
	}))
	events.Emit(key, 42)
}

func scenarios() map[string]func(*trace) {
	return map[string]func(*trace){
		"basic":            basic,
		"duplicate":        duplicate,
		"emit-star":        emitStar,
		"off-unregistered": offUnregistered,
		"symbols":          symbols,
		"wildcard":         wildcard,
	}
}

func scenarioNames() []string {
	all := scenarios()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func run(args []string, stdout, stderr io.Writer) int {
	var scenario func(*trace)
	if len(args) > 0 {
		scenario = scenarios()[args[0]]
	}
	if scenario == nil {
		fmt.Fprintf(stderr, usageFormat, strings.Join(scenarioNames(), "|"))
		return 1
	}
	out := &trace{}
	scenario(out)
	fmt.Fprintln(stdout, out.String())
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
