package mitt_test

import (
	"fmt"

	mitt "github.com/developit/mitt-go"
)

func Example() {
	e := mitt.New[any]()

	// listen to an event
	e.OnFunc(mitt.Key("foo"), func(ev any) { fmt.Println("foo", ev) })

	// listen to all events
	e.OnWildcardFunc(func(t mitt.EventType, ev any) { fmt.Println(t, ev) })

	// fire an event
	e.Emit(mitt.Key("foo"), "bar")

	// clearing all events
	e.All().Clear()

	// working with handler references
	onFoo := e.OnFunc(mitt.Key("foo"), func(any) {})
	e.Off(mitt.Key("foo"), onFoo)

	// Output:
	// foo bar
	// foo bar
}

func ExampleEmitter_Off() {
	e := mitt.New[any]()

	h := e.OnFunc(mitt.Key("foo"), func(any) { fmt.Println("called") })
	e.Emit(mitt.Key("foo"), nil)

	e.Off(mitt.Key("foo"), h)
	e.Emit(mitt.Key("foo"), nil)

	// Output:
	// called
}

func ExampleEmitter_OffAll() {
	e := mitt.New[any]()
	e.OnFunc(mitt.Key("foo"), func(any) { fmt.Println("a") })
	e.OnFunc(mitt.Key("foo"), func(any) { fmt.Println("b") })

	e.OffAll(mitt.Key("foo"))

	// The key survives with an empty handler list.
	hs, ok := e.All().Get(mitt.Key("foo"))
	fmt.Println(ok, len(hs))

	// Output:
	// true 0
}

func ExampleOnTyped() {
	e := mitt.New[any]()

	mitt.OnTyped[any, string](e, mitt.Key("greeting"), func(s string) {
		fmt.Println("hello,", s)
	})
	mitt.EmitTyped[any, string](e, mitt.Key("greeting"), "world")

	// Output:
	// hello, world
}

func ExampleSym() {
	e := mitt.New[any]()

	secret := mitt.Sym("secret")
	e.OnFunc(secret, func(ev any) { fmt.Println("secret:", ev) })

	// A different symbol with the same description is a different event.
	e.Emit(mitt.Sym("secret"), "ignored")
	e.Emit(secret, "delivered")

	// Output:
	// secret: delivered
}

func ExampleEmitter_All() {
	e := mitt.New[any]()

	// Handlers can be installed by mutating the map directly.
	star := mitt.NewWildcardHandler[any](func(t mitt.EventType, ev any) {
		fmt.Println("*", t, ev)
	})
	e.All().Set(mitt.Wildcard, []mitt.Registration[any]{star})

	e.Emit(mitt.Key("foo"), 1)

	// Output:
	// * foo 1
}
