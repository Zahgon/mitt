// EventType is a sealed union (`string | symbol`). Third-party types must not
// be able to satisfy it, which is what guarantees every event key is usable as
// a Go map key.
package main

import mitt "github.com/developit/mitt-go"

type customKey struct{}

func main() {
	var t mitt.EventType = customKey{}
	_ = t
}
