package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDemoScenariosProduceTheDocumentedTrace(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"basic", "foo=1\nremaining=0"},
		{"duplicate", "h=1\nh=1\nleft=1\nh=2"},
		{"emit-star", "w:p|undefined\nw:*|p"},
		{"off-unregistered", "len=1\na=z"},
		{"symbols", "s=42\n*:Symbol(s)=42"},
		{"wildcard", "a=x\n*:a=x\n*:b=y"},
	}

	if len(cases) != len(scenarios()) {
		t.Fatalf("table covers %d scenario(s), binary exposes %d", len(cases), len(scenarios()))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if code := run([]string{tc.name}, &stdout, &stderr); code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
			if got := stdout.String(); got != tc.want+"\n" {
				t.Errorf("stdout = %q, want %q", got, tc.want+"\n")
			}
		})
	}
}

func TestDemoUsageIsWrittenForAnUnknownScenario(t *testing.T) {
	want := "usage: mittdemo <" + strings.Join(scenarioNames(), "|") + ">\n"

	for _, args := range [][]string{nil, {}, {"bogus"}} {
		var stdout, stderr bytes.Buffer

		if code := run(args, &stdout, &stderr); code != 1 {
			t.Errorf("run(%q) exit code = %d, want 1", args, code)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%q) stdout = %q, want empty", args, stdout.String())
		}
		if got := stderr.String(); got != want {
			t.Errorf("run(%q) stderr = %q, want %q", args, got, want)
		}
	}
}

func TestDemoScenarioNamesAreSortedAndComplete(t *testing.T) {
	names := scenarioNames()

	if len(names) != len(scenarios()) {
		t.Fatalf("scenarioNames() = %d name(s), scenarios() = %d", len(names), len(scenarios()))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Errorf("scenarioNames() not sorted at %d: %q >= %q", i, names[i-1], names[i])
		}
	}
	for _, name := range names {
		if scenarios()[name] == nil {
			t.Errorf("scenarioNames() lists %q, which scenarios() does not define", name)
		}
	}
}

func TestDemoShowRendersMissingPayloadAsUndefined(t *testing.T) {
	if got := show(nil); got != "undefined" {
		t.Errorf("show(nil) = %q, want %q", got, "undefined")
	}
	if got := show(42); got != "42" {
		t.Errorf("show(42) = %q, want %q", got, "42")
	}
}
