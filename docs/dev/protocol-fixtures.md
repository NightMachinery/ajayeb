# Protocol fixture storage

The protocol suite retains every named scenario and all three viewers
(`seat0`, `seat1`, `spectator`). No phase, choice, event, redaction or hidden
information example is sampled out to save space.

- `server/internal/engine/testdata/protocol/*.json` records complete views
  and filtered events. `TestProtocolFixtures` compares them to fresh engine
  output and checks coverage of every phase, choice kind and event kind.
  Each top-level view field occupies one line; its nested values stay
  compact. This keeps diffs local without changing the golden values.
  Golden drift diagnostics report a zero-based byte offset and at most
  120 bytes of context on either side instead of dumping whole fields.
- `server/internal/engine/testdata/views/*.json` records a seeded ordinary
  game at fixed points. These remain indented for regression review.
- `server/internal/wire/testdata/messages/*.json` records complete server
  frame shapes, including room metadata. These remain unchanged.
- `web/src/lib/wire/fixtures.gen.ts` retains every recorded frame and every
  protocol view with its existing name. The generator writes repeated large
  view fields once, checks each with `satisfies View["field"]`, and clones
  them into each snapshot. Each complete snapshot still satisfies its wire
  type. The clones let consumers change a fixture independently. The web
  tests compare every generated value to its JSON source and reject shared
  mutable objects.

The 24 protocol files occupy 1,311,172 bytes, down from 2,496,843 bytes
(47% smaller), while making changed fields visible in diffs. The generated
TypeScript occupies 686,096 bytes, down from
3,336,161 bytes (79% smaller). All JSON values and fixture names are kept.

For a readable protocol snapshot, run `python3 -m json.tool <path>`. The
`/dev/fixture` gallery still exposes every viewer of every scenario.

After a deliberate engine change, regenerate the protocol goldens:

```sh
python3 tools/heavy.py run --kind test -- go -C server test ./internal/engine -run '^TestProtocolFixtures$' -update
```

Then run `make gen` to regenerate the TypeScript module. Use the detached
check procedure in the agent guide when working unattended. `make
verify-quick` checks golden drift, generator drift, TypeScript compatibility,
null arrays and equality between every generated snapshot and its source.
Regeneration uses the existing `make gen` target.

## Live version-1 compatibility transcript

`server/internal/protocoltest/testdata/platform-v1-game.jsonl` records a
complete seeded hot-seat game through the real WebSocket, actor and store.
`TestPlatformV1TranscriptBaseline` compares received bytes without parsing,
normalizing identities or re-encoding JSON. Each frame has one separator
newline in the fixture. The synthetic identity, room code, key and setup
are fixed test inputs, never deployment credentials.

The baseline from `46a540d` contains 76 frames (one welcome, 73 states and
two rejects), 69 accepted moves ending in a civilian result, and 2,252,287
bytes including separators. It includes live events, illegal/stale rejects,
a mid-game resync and a final full-history resync. Existing
`TestRecordMessages` goldens still pin created, linked, error and uncommon
state shapes; `TestProtocolFixtures` still pins every named phase and all
three viewers. The live transcript supplements those fixtures rather than
replacing or sampling them.

Only after reviewing an intended protocol change, regenerate this one
transcript with:

```sh
UPDATE_PLATFORM_BASELINES=1 python3 tools/heavy.py run --kind test -- \
  make test-go GO_PKGS=./internal/protocoltest 'GO_RUN=^TestPlatformV1TranscriptBaseline$'
```

Then rerun without the update variable and inspect the diff. This fixture
is not an input to the TypeScript generator, so it does not require editing
existing generated files. See [../testing.md](../testing.md) for the other
M14 baselines and the detached procedure for unattended runs.
