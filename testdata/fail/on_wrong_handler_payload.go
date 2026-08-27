// Mirrors: emitter.on('foo', barHandler) where foo is string and bar is number.
//
//	// @ts-expect-error
//	emitter.on('foo', barHandler);
package main

import mitt "github.com/developit/mitt-go"

func main() {
	e := mitt.New[string]()
	barHandler := func(int) {}
	e.OnFunc(mitt.Key("foo"), barHandler)
}
