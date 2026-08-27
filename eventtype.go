package mitt

import (
	"fmt"
)

const (
	symbolLabel = "Symbol"
	nilLabel    = "<nil>"
	labelFormat = "%s(%s)"
)

// EventType is the Go equivalent of TypeScript's `string | symbol` union used
// for mitt event names.
//
// The interface is *sealed*: it carries an unexported marker method, so the
// only implementations are [Key], [*Symbol] and [*Raw]. This mirrors the closed
// nature of the TS union and guarantees every EventType is usable as a Go map
// key without risking a runtime "unhashable type" panic.
type EventType interface {
	eventType() eventKind
}

// eventKind discriminates the closed set of EventType implementations.
type eventKind uint8

const (
	kindKey eventKind = iota
	kindSymbol
	kindRaw
)

// Key is a string event name, equivalent to the `string` half of TS's
// `EventType = string | symbol`.
//
// Keys are compared byte-for-byte. mitt performs no normalisation whatsoever:
// Key("FOO"), Key("Foo") and Key("foo") are three distinct events.
type Key string

func (Key) eventType() eventKind { return kindKey }

// String returns the underlying string.
func (k Key) String() string { return string(k) }

// Wildcard is the special event name `'*'`. Handlers registered under it are
// invoked for every emitted event, after the type-matched handlers.
const Wildcard Key = "*"

// Symbol is the Go equivalent of a JavaScript `Symbol` used as an event name.
//
// Like JS symbols, identity is by reference, not by description: two symbols
// created with the same description are different event keys.
//
//	a := mitt.Sym("evt")
//	b := mitt.Sym("evt")
//	a == b // false
type Symbol struct {
	desc string
}

// Sym creates a new unique Symbol with the given (purely informational)
// description. Equivalent to JS `Symbol(desc)`.
func Sym(desc string) *Symbol { return &Symbol{desc: desc} }

func (*Symbol) eventType() eventKind { return kindSymbol }

// Description returns the symbol's description, equivalent to JS
// `symbol.description`.
func (s *Symbol) Description() string { return s.desc }

// String implements fmt.Stringer, mirroring JS `String(symbol)`.
func (s *Symbol) String() string {
	desc := nilLabel
	if s != nil {
		desc = s.desc
	}
	return fmt.Sprintf(labelFormat, symbolLabel, desc)
}

// Raw boxes an arbitrary event payload that has been handed to a
// [WildcardHandler] in its *type* position.
//
// This only happens when reproducing one specific JavaScript behaviour: calling
// Emit with the Wildcard key. In JS, `emit('*', evt)` first treats the `'*'`
// handler list as ordinary handlers and calls each with `handler(evt)`, so a
// two-argument wildcard handler receives `(evt, undefined)` — the payload lands
// in the type parameter. Go is statically typed and cannot pass an arbitrary
// payload where an EventType is expected, so the payload is boxed in a Raw.
//
// See DESIGN.md, "Arity coercion".
type Raw struct {
	v any
}

func (*Raw) eventType() eventKind { return kindRaw }

// Value returns the boxed payload.
func (r *Raw) Value() any {
	if r == nil {
		return nil
	}
	return r.v
}

// String implements fmt.Stringer by rendering the boxed payload directly.
//
// The box is deliberately transparent. JavaScript has no box: `emit('*', evt)`
// invokes a wildcard handler as `handler(evt)`, so its type parameter *is* the
// payload and `String(type)` prints the payload. Wrapping the output in
// `Raw(...)` would leak a Go-only construct into user-visible text.
func (r *Raw) String() string {
	if r == nil {
		return nilLabel
	}
	return sprint(r.v)
}
