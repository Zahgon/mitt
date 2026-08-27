// Runs every conformance program against the REAL mitt source
// (scraped repos/TS/mitt/src/index.ts, executed unmodified by Node's type
// stripping) and records a canonical trace.
import { readFileSync, writeFileSync } from 'node:fs';
import mitt from './mitt.ts';

const programs = JSON.parse(readFileSync(process.argv[2], 'utf8'));

function runProgram(prog) {
  const emitter = mitt();
  const trace = [];

  const symbols = new Map();
  const symbolNames = new Map();
  const symbolFor = (name) => {
    if (!symbols.has(name)) {
      const s = Symbol(name);
      symbols.set(name, s);
      symbolNames.set(s, name);
    }
    return symbols.get(name);
  };

  const decodeType = (t) => (t.t === 'sym' ? symbolFor(t.v) : t.v);
  const decodeValue = (v) => {
    if (v.t === 'undef') return undefined;
    if (v.t === 'sym') return symbolFor(v.v);
    return v.v;
  };

  const encode = (v) => {
    if (v === undefined) return { t: 'undef' };
    if (typeof v === 'symbol') return { t: 'sym', v: symbolNames.get(v) ?? v.description };
    if (typeof v === 'number') return { t: 'num', v };
    if (typeof v === 'string') return { t: 'str', v };
    return { t: 'other', v: String(v) };
  };

  const handlers = new Map();
  const handlerFor = (name, kind) => {
    if (!handlers.has(name)) {
      const fn =
        kind === 'wild'
          ? (type, evt) => { trace.push({ h: name, args: [encode(type), encode(evt)] }); }
          : (evt) => { trace.push({ h: name, args: [encode(evt)] }); };
      fn.__name = name;
      handlers.set(name, fn);
    }
    return handlers.get(name);
  };

  const snapshot = () =>
    [...emitter.all.entries()].map(([k, v]) => ({
      type: encode(k),
      handlers: v.map((f) => f.__name),
    }));

  for (const op of prog.ops) {
    switch (op.op) {
      case 'on':
        emitter.on(decodeType(op.type), handlerFor(op.h, op.kind));
        break;
      case 'off':
        if (op.h === null) emitter.off(decodeType(op.type));
        else emitter.off(decodeType(op.type), handlerFor(op.h, 'single'));
        break;
      case 'emit':
        emitter.emit(decodeType(op.type), decodeValue(op.payload));
        break;
      case 'state':
        trace.push({ state: snapshot() });
        break;
      default:
        throw new Error(`unknown op ${op.op}`);
    }
  }
  return trace;
}

const expected = {};
for (const prog of programs) expected[prog.name] = runProgram(prog);

writeFileSync(process.argv[3], JSON.stringify(expected, null, 2) + '\n');
console.log(`traced ${programs.length} programs`);
