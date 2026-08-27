// Mirrors: emitter.emit('foo', 1) where foo is a string event.
//
//	// @ts-expect-error
//	emitter.emit('foo', 1);
package main

import mitt "github.com/developit/mitt-go"

func main() {
	e := mitt.New[string]()
	e.Emit(mitt.Key("foo"), 1)
}
