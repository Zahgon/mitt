# Porting notes

How mitt v3.0.1 maps onto Go, and why each decision was made. The source is 123
lines of TypeScript; almost all of the work here is preserving semantics that
JavaScript provides for free.

## Handler identity

The original removes a listener by reference:

```js
off(type, handler) {
  const handlers = all.get(type);
  if (handlers) {
    if (handler) handlers.splice(handlers.indexOf(handler) >>> 0, 1);
    else all.set(type, []);
  }
}
```

`indexOf` relies on `===` over function references. Go has no equivalent: funcs
are not comparable, and `==` on two `func` values is a compile error.

Three options were considered.

1. **`reflect.ValueOf(fn).Pointer()`** — wrong. Every closure produced from the
   same function literal shares one code pointer, so two distinct listeners
   created in a loop would compare equal and `Off` would remove the wrong one.
   `TestClosuresFromSameLiteralAreDistinctRegistrations` fails under this scheme.
2. **Caller-supplied IDs** — changes the API shape and pushes bookkeeping onto
   the user.
3. **Pointer handles** — chosen. `On`/`OnFunc` wrap the func in a heap-allocated
   `*Handler[E]` and return it. Pointer equality then reproduces JS reference
   equality exactly, and the returned handle reads like the function reference
   the original passes around.

`Registration[E]` is a sealed interface (unexported marker method) with two
implementations, `*Handler[E]` (one argument) and `*WildcardHandler[E]` (two).
Sealing it means dispatch can exhaustively handle both without a default case
that could silently swallow a third kind.

## Event types

`type EventType = string | symbol` becomes a sealed interface with an unexported
marker method:

-   `Key string` — string events. `Wildcard` is `Key("*")`.
-   `*Symbol` — created by `Sym(description)`. Pointer identity gives the
    uniqueness JS symbols have; two `Sym("x")` values are different events.
-   `*Raw` — an internal box, described below.

Sealing matters for a second reason: an event type is used as a Go map key, so
it must be hashable. If arbitrary user types could implement `EventType`, a
struct containing a slice would compile and then panic at runtime on insert.
`testdata/fail/sealed_eventtype.go` pins this.

## Arity coercion

JavaScript silently truncates and pads arguments. Two consequences show up in
mitt, both confirmed against Node before being encoded here.

A **one-argument handler registered on `'*'`** is called as `handler(type, evt)`
and therefore receives the *type*:

```js
emitter.on('*', e => console.log(e));
emitter.emit('foo', 'PAYLOAD');   // logs 'foo'
```

`Handler.invokeWildcard` reproduces this by converting the event type into the
payload type `E` (`asPayload`): a direct type assertion when `E` admits it, then
`Key`→`string`, then unwrapping a `*Raw`, otherwise the zero value.

A **two-argument handler reached through the type lookup** — which happens when
you literally `emit('*', payload)` — is called as `handler(evt)`, so its first
parameter receives the payload and its second is `undefined`.
`WildcardHandler.invokeTyped` reproduces this via `asEventType`, which is where
`*Raw` comes from: the payload has landed in the event-type position, and if it
is neither an `EventType` nor a string it gets boxed so it can be passed through
without losing its value. `Raw.Value()` unwraps it.

This is the only reason `*Raw` exists. It is unreachable unless you emit the
wildcard key itself.

## The handler map

`emitter.all` is a JS `Map`, which iterates in insertion order. Go maps
deliberately randomise iteration, and mitt's own test suite inspects `all`
directly, so a plain map would produce an emitter that is observably different.

`HandlerMap[E]` therefore keeps parallel `keys []EventType`, `index
map[EventType]int` and `vals [][]Registration[E]` slices. `Delete` removes from
the ordered slices and re-indexes the remainder. The map stays fully live and
mutable, because the original documents `all` as a supported extension point.

`Range` iterates over a snapshot of the keys so that a callback may mutate the
map without invalidating the iteration.

## Dispatch

```js
emit(type, evt) {
  let handlers = all.get(type);
  if (handlers) handlers.slice().map(handler => { handler(evt); });

  handlers = all.get('*');
  if (handlers) handlers.slice().map(handler => { handler(type, evt); });
}
```

Two details are load-bearing and easy to lose in a port:

-   **`.slice()`** copies the list before dispatch. A handler added during an
    emit does not fire; a handler removed during an emit still does. `Emit`
    snapshots both lists for the same reason.
-   **The wildcard lookup happens after the typed handlers have run.** A
    wildcard handler registered by one of those handlers *does* fire in the same
    emit. Reordering the two lookups for tidiness would break this;
    `TestWildcardListLookedUpAfterTypedDispatch` guards it.

Panics propagate, matching a thrown exception aborting the remaining dispatch.

## Optional payloads

`emit(type)` with no event passes `undefined`. `EmitVoid` is the equivalent and
delivers the zero value of `E`. For `OnTyped`, a nil payload yields the zero
value of `T` rather than a panic, since "no payload" is not a type error.

## Type-system parity gaps

Two differences are forced by Go's type system and cannot be closed inside the
library. Both are behaviour-visible, so they are recorded here rather than left
for a user to discover.

**`null` and `undefined` collapse into one `nil`.** JavaScript has two distinct
empty values and mitt passes whichever it is straight through, so a handler can
tell `emit('p', null)` from `emit('p')`. Go's `any` has a single nil, so
`Emit(t, nil)` and `EmitVoid(t)` deliver a value the handler cannot tell apart.

The library loses nothing of its own: payloads are opaque to mitt and are
forwarded byte-for-byte, so a caller who needs the distinction models it the way
Go normally does — with a sentinel, a pointer, or a wrapper type:

```go
type Null struct{}

e.Emit(mitt.Key("p"), Null{}) // the JS `null` case
e.EmitVoid(mitt.Key("p"))     // the JS `undefined` case
```

**Per-key payload types are not expressible.** TypeScript's
`Record<EventType, unknown>` maps every event name to its own payload type, and
`on`/`emit` are checked against that mapping. Go generics have no equivalent of
an indexed mapped type, so `Emitter[E]` fixes one payload type for the whole
emitter. Most callers use `Emitter[any]`; `OnTyped` and `EmitTyped` recover
static typing for an individual event where it matters.

## Concurrency

The original is single-threaded and unsynchronised, so `New` is too — adding
locks would change performance characteristics for every user to solve a problem
JS never has.

`NewSync` is the opt-in variant. It snapshots the handler lists under a lock and
then releases it before invoking anything, so a handler can call `On`, `Off` or
`Emit` without deadlocking. `TestSyncEmitterHandlerReentrancyDoesNotDeadlock`
covers that path.

## Test strategy

-   `mitt_test.go` — the 20 mocha specs from `test/index_test.ts`, ported 1:1
    with a `spy` helper standing in for sinon.
-   `invariants_test.go` — the quirks above, each one first reproduced by running
    the original JavaScript under Node.
-   `typecheck_test.go` plus `testdata/fail/` — the port of
    `test/test-types-compilation.ts`. Each fixture must fail to compile, which is
    the Go analogue of `@ts-expect-error`.
-   `concurrency_test.go` — run under `-race`; also asserts that the sync and
    plain emitters agree on ordering.
