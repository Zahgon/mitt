<p align="center">
  <img src="https://i.imgur.com/BqsX9NT.png" width="300" height="300" alt="mitt">
</p>

# mitt-go

> Tiny functional event emitter / pubsub — a Go port of [mitt](https://github.com/developit/mitt).

-   **Microscopic:** the entire emitter is a few hundred lines with no dependencies
-   **Useful:** a wildcard `"*"` event type listens to all events
-   **Familiar:** same names & ideas as the original `mitt`
-   **Functional:** methods don't rely on hidden global state
-   **Faithful:** behaviour is verified against the JavaScript original, quirks included

This is a complete, functionally equivalent port of mitt v3.0.1. Every observable
behaviour of the original — including its surprising edge cases — is reproduced
and pinned by a test. See [DESIGN.md](DESIGN.md) for the porting decisions.

## Table of Contents

-   [Install](#install)
-   [Usage](#usage)
-   [API](#api)
-   [Differences from the JavaScript original](#differences-from-the-javascript-original)
-   [Behavioural parity](#behavioural-parity)
-   [Contribute](#contribute)
-   [License](#license)

## Install

```sh
go get github.com/developit/mitt-go
```

```go
import mitt "github.com/developit/mitt-go"
```

Requires Go 1.21 or newer (generics and `any`).

## Usage

```go
e := mitt.New[any]()

// listen to an event
e.OnFunc(mitt.Key("foo"), func(ev any) { fmt.Println("foo", ev) })

// listen to all events
e.OnWildcardFunc(func(t mitt.EventType, ev any) { fmt.Println(t, ev) })

// fire an event
e.Emit(mitt.Key("foo"), map[string]string{"a": "b"})

// clearing all events
e.All().Clear()

// working with handler references
onFoo := e.OnFunc(mitt.Key("foo"), func(any) {}) // listen
e.Off(mitt.Key("foo"), onFoo)                    // unlisten
```

### Typed events

The original library gets its type safety from a TypeScript `Events` map. In Go
the equivalent is either a concrete payload type on the emitter:

```go
e := mitt.New[string]()
e.OnFunc(mitt.Key("greeting"), func(s string) { fmt.Println("hello,", s) })
e.Emit(mitt.Key("greeting"), "world")
```

…or per-event payload assertions on a heterogeneous emitter:

```go
e := mitt.New[any]()

mitt.OnTyped[any, LoginEvent](e, mitt.Key("login"), func(ev LoginEvent) {
    fmt.Println(ev.User)
})
mitt.EmitTyped[any, LoginEvent](e, mitt.Key("login"), LoginEvent{User: "jack"})
```

`OnTyped` panics if a payload of the wrong type is delivered, which is the
closest runtime analogue to TypeScript rejecting the call at compile time.

### Symbol keys

JavaScript symbols are unique by reference, not by description. `mitt.Sym`
reproduces that: two symbols with the same description are different events.

```go
secret := mitt.Sym("secret")
e.OnFunc(secret, func(ev any) { fmt.Println(ev) })

e.Emit(mitt.Sym("secret"), "ignored")   // different symbol, no handler runs
e.Emit(secret, "delivered")
```

### Concurrency

`New` matches the original exactly: no synchronisation, like single-threaded JS.
If you need to share an emitter across goroutines, use `NewSync`, which
serialises registry access and dispatches outside the lock so handlers may
safely re-enter the emitter.

```go
e := mitt.NewSync[any]()
```

## API

| mitt (TS)                     | mitt-go                                        |
| ----------------------------- | ---------------------------------------------- |
| `mitt<Events>()`              | `mitt.New[E]()`                                 |
| `mitt<Events>(all)`           | `mitt.NewWith[E](all)`                          |
| `emitter.all`                 | `e.All() *HandlerMap[E]`                        |
| `emitter.on(type, handler)`   | `e.On(type, h)` / `e.OnFunc(type, fn)`          |
| `emitter.on('*', handler)`    | `e.OnWildcard(h)` / `e.OnWildcardFunc(fn)`      |
| `emitter.off(type, handler)`  | `e.Off(type, h)`                                |
| `emitter.off(type)`           | `e.OffAll(type)`                                |
| `emitter.emit(type, evt)`     | `e.Emit(type, evt)`                             |
| `emitter.emit(type)`          | `e.EmitVoid(type)`                              |
| `emitter.all.clear()`         | `e.All().Clear()`                               |
| `type EventType = string \| symbol` | `mitt.EventType` (`Key` \| `*Symbol` \| `*Raw`) |
| `'*'`                         | `mitt.Wildcard`                                 |

`On`, `OnFunc`, `OnWildcard` and `OnWildcardFunc` return the registration handle
you pass back to `Off`.

## Differences from the JavaScript original

These are consequences of the target language, not behaviour changes.

-   **Handlers are handles, not bare funcs.** JS removes listeners by reference
    identity; Go funcs are not comparable. `OnFunc` returns a `*Handler[E]` that
    plays the role of the function reference. See
    [DESIGN.md](DESIGN.md#handler-identity).
-   **Event keys are a sealed union.** `EventType` is `Key` (string) or
    `*Symbol`, mirroring `string | symbol`. Outside types cannot implement it,
    which guarantees every key is a valid Go map key.
-   **`all` is `*HandlerMap[E]`, not a raw map.** A JS `Map` preserves insertion
    order and Go maps do not, so the port keeps an ordered structure. It exposes
    `Get`/`Set`/`Has`/`Delete`/`Len`/`Clear`/`Keys`/`Range` and is fully live:
    mutating it changes emitter behaviour, exactly as in the original.
-   **A throwing handler is a panicking handler.** It propagates to the caller of
    `Emit` and aborts the remaining dispatch, matching JS exception semantics.
-   **`New` is not goroutine-safe**, because the original isn't either. `NewSync`
    is the opt-in concurrent variant.

## Behavioural parity

Each of these was confirmed by running the original JavaScript, then pinned by a
test in `invariants_test.go`.

| Behaviour | Notes |
| --- | --- |
| `off` with an unregistered handler | Silent no-op. In JS `indexOf` returns `-1` and `-1 >>> 0` is `4294967295`, so the `splice` is out of range and removes nothing. A naive port removes the last handler instead. |
| `off(type)` with no handler | Sets the entry to an empty list; the key remains present. |
| `off` on an unknown type | Does not create an entry. |
| Duplicate registration | Allowed; the handler fires once per registration. |
| `off` with duplicates | Removes only the first occurrence. |
| Dispatch order | All handlers for the type, then all wildcard handlers. |
| Handler list snapshot | The type's list is copied before dispatch, so handlers added during an emit do not fire, and handlers removed during an emit still do. |
| Wildcard lookup timing | The wildcard list is read *after* the typed handlers run, so a wildcard handler registered mid-dispatch does fire in the same emit. |
| `emit('*', payload)` | Invokes a wildcard handler twice: once via the type lookup, once via the wildcard lookup. |
| One-argument handler on `'*'` | Receives the event type, not the payload — JS truncates extra arguments. |
| Event keys | Compared exactly; no case normalisation. |
| Symbols | Unique by identity, never by description. |
| Unknown type | Emitting is a no-op. |
| Dispatch | Fully synchronous. |

## Contribute

```sh
go test ./...          # run the suite
go test ./... -race    # race detector
make check             # fmt, vet, lint, race, coverage
```

`testdata/fail` holds programs that must **not** compile; they are the port of
mitt's `@ts-expect-error` type tests and are verified by `typecheck_test.go`.

## License

MIT © [Jason Miller](https://github.com/developit) — see [LICENSE](LICENSE).
The Go port preserves the original copyright.
