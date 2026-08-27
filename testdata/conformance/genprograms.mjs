// Emits testdata/conformance/programs.json — the shared op-scripts that both
// the original mitt and the Go port are driven through.
import { writeFileSync } from 'node:fs';

const S = (v) => ({ t: 'str', v });
const SYM = (v) => ({ t: 'sym', v });
const NUM = (v) => ({ t: 'num', v });
const UNDEF = { t: 'undef' };
const STAR = S('*');

const on = (type, h) => ({ op: 'on', type, h, kind: 'single' });
const onw = (type, h) => ({ op: 'on', type, h, kind: 'wild' });
const off = (type, h) => ({ op: 'off', type, h });
const offall = (type) => ({ op: 'off', type, h: null });
const emit = (type, payload) => ({ op: 'emit', type, payload });
const emitv = (type) => ({ op: 'emit', type, payload: UNDEF });
const state = () => ({ op: 'state' });

const programs = [
  {
    name: 'on-emit-basic',
    ops: [on(S('foo'), 'h1'), state(), emit(S('foo'), S('bar')), state()],
  },
  {
    name: 'emit-unknown-type-noop',
    ops: [on(S('foo'), 'h1'), emit(S('nope'), NUM(1)), state()],
  },
  {
    name: 'multiple-handlers-order',
    ops: [
      on(S('foo'), 'h1'), on(S('foo'), 'h2'), on(S('foo'), 'h3'),
      emit(S('foo'), NUM(7)), state(),
    ],
  },
  {
    name: 'duplicate-registration-fires-twice',
    ops: [on(S('foo'), 'h1'), on(S('foo'), 'h1'), emit(S('foo'), S('x')), state()],
  },
  {
    name: 'off-removes-first-duplicate-only',
    ops: [
      on(S('foo'), 'h1'), on(S('foo'), 'h1'), off(S('foo'), 'h1'),
      state(), emit(S('foo'), S('x')),
    ],
  },
  {
    name: 'off-unregistered-handler-is-noop',
    ops: [
      on(S('foo'), 'h1'), on(S('foo'), 'h2'), off(S('foo'), 'hNever'),
      state(), emit(S('foo'), NUM(1)),
    ],
  },
  {
    name: 'off-unknown-type-does-not-create-entry',
    ops: [off(S('ghost'), 'h1'), offall(S('ghost2')), state()],
  },
  {
    name: 'offall-clears-to-empty-keeps-key',
    ops: [
      on(S('foo'), 'h1'), on(S('foo'), 'h2'), offall(S('foo')),
      state(), emit(S('foo'), NUM(1)),
    ],
  },
  {
    name: 'off-then-reregister',
    ops: [
      on(S('foo'), 'h1'), off(S('foo'), 'h1'), state(),
      on(S('foo'), 'h1'), state(), emit(S('foo'), S('again')),
    ],
  },
  {
    name: 'wildcard-receives-type-and-payload',
    ops: [
      on(S('foo'), 'h1'), onw(STAR, 'w1'),
      emit(S('foo'), S('p1')), emit(S('bar'), NUM(2)), state(),
    ],
  },
  {
    name: 'wildcard-only',
    ops: [onw(STAR, 'w1'), emit(S('anything'), S('p')), emitv(S('other')), state()],
  },
  {
    name: 'dispatch-order-typed-before-wildcard',
    ops: [
      onw(STAR, 'w1'), on(S('foo'), 'h1'), onw(STAR, 'w2'), on(S('foo'), 'h2'),
      emit(S('foo'), NUM(9)), state(),
    ],
  },
  {
    name: 'emit-wildcard-key-double-invokes',
    ops: [onw(STAR, 'w1'), emit(STAR, NUM(1)), state()],
  },
  {
    name: 'emit-wildcard-key-string-payload',
    ops: [onw(STAR, 'w1'), emit(STAR, S('pay')), state()],
  },
  {
    name: 'emit-wildcard-key-undefined-payload',
    ops: [onw(STAR, 'w1'), emitv(STAR), state()],
  },
  {
    name: 'single-arg-handler-on-wildcard-gets-type',
    ops: [on(STAR, 'h1'), emit(S('foo'), S('PAYLOAD')), emit(S('bar'), NUM(3)), state()],
  },
  {
    name: 'single-arg-handler-on-wildcard-symbol-type',
    ops: [on(STAR, 'h1'), emit(SYM('A'), S('PAYLOAD')), state()],
  },
  {
    name: 'wildcard-off',
    ops: [
      onw(STAR, 'w1'), onw(STAR, 'w2'), off(STAR, 'w1'),
      emit(S('foo'), NUM(1)), state(),
    ],
  },
  {
    name: 'wildcard-offall',
    ops: [onw(STAR, 'w1'), offall(STAR), emit(S('foo'), NUM(1)), state()],
  },
  {
    name: 'symbol-keys-identity',
    ops: [
      on(SYM('A'), 'h1'), on(SYM('B'), 'h2'),
      emit(SYM('A'), S('a')), emit(SYM('B'), S('b')), state(),
    ],
  },
  {
    name: 'symbol-same-description-distinct',
    ops: [on(SYM('A'), 'h1'), emit(SYM('A2'), S('nope')), emit(SYM('A'), S('yes')), state()],
  },
  {
    name: 'symbol-and-string-coexist',
    ops: [
      on(SYM('A'), 'h1'), on(S('A'), 'h2'),
      emit(SYM('A'), NUM(1)), emit(S('A'), NUM(2)), state(),
    ],
  },
  {
    name: 'case-sensitivity',
    ops: [
      on(S('foo'), 'h1'), on(S('FOO'), 'h2'), on(S('Foo'), 'h3'),
      emit(S('foo'), NUM(1)), emit(S('FOO'), NUM(2)), emit(S('Foo'), NUM(3)),
      off(S('FOO'), 'h1'), state(),
    ],
  },
  {
    name: 'exotic-string-keys',
    ops: [
      on(S('constructor'), 'h1'), on(S('__proto__'), 'h2'), on(S(''), 'h3'),
      on(S('hasOwnProperty'), 'h4'), on(S('toString'), 'h5'), on(S('日本語 🎉'), 'h6'),
      emit(S('constructor'), NUM(1)), emit(S('__proto__'), NUM(2)),
      emit(S(''), NUM(3)), emit(S('hasOwnProperty'), NUM(4)),
      emit(S('toString'), NUM(5)), emit(S('日本語 🎉'), NUM(6)),
      state(),
    ],
  },
  {
    name: 'insertion-order-preserved',
    ops: [
      on(S('z'), 'h1'), on(S('a'), 'h2'), on(S('m'), 'h3'),
      on(S('b'), 'h4'), on(S('y'), 'h5'), state(),
      offall(S('m')), on(S('c'), 'h6'), state(),
    ],
  },
  {
    name: 'undefined-payload',
    ops: [on(S('foo'), 'h1'), onw(STAR, 'w1'), emitv(S('foo')), state()],
  },
  {
    name: 'number-payloads',
    ops: [
      on(S('n'), 'h1'), onw(STAR, 'w1'),
      emit(S('n'), NUM(0)), emit(S('n'), NUM(-1)), emit(S('n'), NUM(2147483647)),
      state(),
    ],
  },
  {
    name: 'many-events-interleaved',
    ops: [
      on(S('a'), 'h1'), on(S('b'), 'h2'), onw(STAR, 'w1'), on(S('a'), 'h3'),
      emit(S('a'), NUM(1)), off(S('a'), 'h1'), emit(S('a'), NUM(2)),
      offall(S('b')), emit(S('b'), NUM(3)), on(S('b'), 'h4'),
      emit(S('b'), NUM(4)), off(STAR, 'w1'), emit(S('a'), NUM(5)),
      state(),
    ],
  },
  {
    name: 'handler-shared-across-types',
    ops: [
      on(S('a'), 'h1'), on(S('b'), 'h1'), off(S('a'), 'h1'),
      emit(S('a'), NUM(1)), emit(S('b'), NUM(2)), state(),
    ],
  },
  {
    name: 'wildcard-handler-also-on-typed',
    ops: [
      onw(S('foo'), 'w1'), onw(STAR, 'w1'),
      emit(S('foo'), S('p')), state(),
    ],
  },
  {
    name: 'emit-before-any-registration',
    ops: [emit(S('foo'), NUM(1)), emitv(STAR), state(), on(S('foo'), 'h1'), emit(S('foo'), NUM(2))],
  },
  {
    name: 'repeated-offall-idempotent',
    ops: [
      on(S('foo'), 'h1'), offall(S('foo')), offall(S('foo')), offall(S('foo')),
      state(),
    ],
  },
];

writeFileSync(process.argv[2], JSON.stringify(programs, null, 2) + '\n');
console.log(`wrote ${programs.length} programs`);
