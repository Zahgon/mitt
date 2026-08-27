// A Handler built for one payload type cannot be registered on an emitter with
// a different payload type. TypeScript enforces the same through Events[Key].
package main

import mitt "github.com/developit/mitt-go"

func main() {
	e := mitt.New[string]()
	h := mitt.NewHandler[int](func(int) {})
	e.On(mitt.Key("foo"), h)
}
