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

`server/internal/protocoltest/testdata/platform-v1-game.jsonl` is a compact
SHA-256 manifest of a complete seeded hot-seat game through the real
WebSocket, actor and store. Its first line pins the stream's frame count,
length and SHA-256; each following short line pins one frame's zero-based
index, kind, length and SHA-256. Frame lengths exclude separators. The
stream has one newline after each untouched received frame. No fields,
whitespace or field order are normalized before hashing.

The baseline from `46a540d` contains 76 frames (one welcome, 73 states and
two rejects), 69 accepted moves ending in a civilian result, and 2,252,287
stream bytes including separators. The manifest occupies 9,128 bytes in
77 lines, replacing the full 2,252,287-byte recording. It covers live
events, illegal/stale rejects, a mid-game resync and final history resync.
The synthetic identity, room code, key and setup are fixed test inputs,
never deployment credentials.

`TestPlatformV1TranscriptBaseline` compares every frame record and the
whole stream digest. On mismatch it names the first differing frame with
expected/actual lengths and digests, saves the actual stream in a persistent
OS temporary file, and prints a master-capture command and byte-diff
command. The capture command archives only master's `server/` sources and
injects this capture test into that temporary copy. It works before the
test is merged into master, changes no checkout or ref, and cleans the
source copy on exit. The actual and reference stream files remain available
for inspection. Missing/extra frames and stream-only metadata differences
also have bounded diagnostics, without dumping frames into the test log.

Readable shapes stay in the existing `TestRecordMessages` and
`TestProtocolFixtures` goldens, including additional message shapes,
every named phase and all three viewers. No readable coverage is sampled
out or duplicated into this manifest.

After reviewing an intended protocol change, regenerate the manifest:

```sh
UPDATE_PLATFORM_BASELINES=1 python3 tools/heavy.py run --kind test -- \
  make test-go GO_PKGS=./internal/protocoltest 'GO_RUN=^TestPlatformV1TranscriptBaseline$'
```

To capture this checkout's full stream explicitly, without updating the
manifest or comparing it in this capture-only invocation:

```sh
PLATFORM_V1_STREAM_PATH=/tmp/ajayeb-platform-v1-current.jsonl \
  python3 tools/heavy.py run --kind test -- \
  go -C server test ./internal/protocoltest -run '^TestPlatformV1TranscriptBaseline$' -count=1
```

Normal verification leaves both variables unset. This manifest is not an
input to the TypeScript generator. See [../testing.md](../testing.md) for
the other M14 baselines and use the detached guard procedure for unattended
runs.
