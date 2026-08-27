# Conformance fixtures

Differential test data. `expected.json` is a recording of what the **original**
mitt does, and `conformance_test.go` requires the Go port to reproduce it
exactly.

| File | Role |
| --- | --- |
| `programs.json` | Op-scripts run against both implementations. |
| `expected.json` | Traces recorded from the original `src/index.ts`. |
| `genprograms.mjs` | Regenerates `programs.json`. |
| `driver.mjs` | Runs the programs through the original and writes `expected.json`. |

A trace records every handler invocation with its exact arguments, plus the full
contents of `all` at each `state` checkpoint — so it captures dispatch order,
argument coercion, handler-list contents and key insertion order.

## Regenerating

Node 22.6+ runs mitt's TypeScript source directly via type stripping, so the
recording is made against the unmodified original rather than a reimplementation.

```sh
cp /path/to/mitt/src/index.ts ./mitt.ts
node genprograms.mjs programs.json
node driver.mjs programs.json expected.json
rm mitt.ts
```

`mitt.ts` is intentionally not committed; it belongs to the upstream project.

## Adding a case

Add it to `genprograms.mjs`, regenerate both files, then run `go test ./...`.
Never hand-edit `expected.json` — its only authority is the original library.

Payloads are limited to strings, numbers, symbols and `undefined` so that both
runtimes can encode them identically.
