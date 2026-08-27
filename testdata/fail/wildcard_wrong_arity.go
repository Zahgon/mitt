// Mirrors: emitter.on('*', barHandler) — a handler whose parameter type does
// not accept the event key.
//
//	// @ts-expect-error
//	emitter.on('*', barHandler);
package main

import mitt "github.com/developit/mitt-go"

func main() {
	e := mitt.New[string]()
	e.OnWildcard(mitt.NewWildcardHandler[string](func(int, string) {}))
}
