package mitt

import "testing"

func TestSealedEventTypeKindsAreDistinctAndNilSafe(t *testing.T) {
	var (
		nilSymbol *Symbol
		nilRaw    *Raw
	)

	cases := []struct {
		name string
		et   EventType
		want eventKind
	}{
		{"key", Key("foo"), kindKey},
		{"wildcard", Wildcard, kindKey},
		{"symbol", Sym("desc"), kindSymbol},
		{"nil symbol", nilSymbol, kindSymbol},
		{"raw", &Raw{v: 1}, kindRaw},
		{"nil raw", nilRaw, kindRaw},
	}

	seen := map[eventKind]bool{}
	for _, tc := range cases {
		if got := tc.et.eventType(); got != tc.want {
			t.Errorf("%s: eventType() = %d, want %d", tc.name, got, tc.want)
		}
		seen[tc.want] = true
	}

	if len(seen) != 3 {
		t.Fatalf("distinct event kinds exercised: got %d, want 3", len(seen))
	}
}

func TestSealedRegistrationArityIsNilSafe(t *testing.T) {
	var (
		nilHandler  *Handler[int]
		nilWildcard *WildcardHandler[int]
	)

	cases := []struct {
		name string
		reg  Registration[int]
		want arity
	}{
		{"handler", NewHandler(func(int) {}), arityTyped},
		{"nil handler", nilHandler, arityTyped},
		{"wildcard handler", NewWildcardHandler(func(EventType, int) {}), arityWildcard},
		{"nil wildcard handler", nilWildcard, arityWildcard},
	}

	for _, tc := range cases {
		if got := tc.reg.registration(); got != tc.want {
			t.Errorf("%s: registration() = %d, want %d", tc.name, got, tc.want)
		}
	}
}
