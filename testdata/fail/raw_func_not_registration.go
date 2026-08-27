// A bare func is not a Registration: Off needs reference identity, so handlers
// must be wrapped in a handle. See DESIGN.md, "Handler identity".
package main

import mitt "github.com/developit/mitt-go"

func main() {
	e := mitt.New[any]()
	e.On(mitt.Key("foo"), func(any) {})
}
