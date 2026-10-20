# Master e2e failure attribution

## Persian Continue age label

`65d3937` (M11.wording-consistency, October 18) changed Home to use the
shared `ageLabel` helper. In Persian, Age II is written as `دوران دوم`.
The returning-device hydration test still expected the digit `۲`, so the
test was stale. It now checks the actual localized age wording while
retaining its hydration, theme, tutorial-offer and saved-room assertions.
The comparison ignores invisible bidi isolation marks, as the wording
consistency tests already do.

## Live bot moves followed by advisor snapshots

`f32187d` (M11.advisor-strength) moved the default move review off the room
actor and added later state frames containing the result. The server sends
those frames at the same game version with no new events. Animation case c
exposed a presenter bug: a queued advisor frame counted as move backlog,
so the live bot move just before it lost its animation. A later cosmetic
frame could also finish the tail of a move already playing.

The presenter now coalesces queued same-version cosmetic data with the
live move's events. `shownEvents` retains the displayed move for live-region
announcements through cosmetic refreshes and resets on clear or a new
non-cosmetic position. A cosmetic update to the displayed position updates
the view without capturing or replaying effects. The newest authoritative
frame keeps its identity; a separate effect source preserves the live
move's events for animation and sound. Frames carrying history
still reset effects, and real move backlog still catches up. Unit tests
cover both paths; case c requires a timed animation sample for every bot
reply it exercises, in addition to its existing 450 ms limit.

## Long-task acceptance investigation

The animation performance tests now respect the configured retry policy.
Their previous local override retried twice even when the command requested
`--retries=0`. Unprofiled runs retain their room/task/animation records for
attribution. Diagnostic profiling and first-attempt acceptance remain
separate, and the 50 ms threshold and animation windows are unchanged.

The reproduced diagnostic task is the native age-summary `showModal()` call
while age-end coins are still moving. Most of its cost is style recalculation,
also present at the game-over modal. CSS and HTML inert icon experiments did
not reduce it. Hiding the sprite broke five clipped icons, so that change was
rejected. The client now renders the same original shapes directly, with local
clipping ids, removing the sprite and SVG use shadow trees. The first direct-rendering
diagnostic restyles 1,870 elements instead of 2,632 on desktop, and 1,183
instead of 1,747 on mobile, before the later wrapper reductions. The fixed-scale pixel comparison checks all
81 icons in both themes and with inherited two-tone paints. Native modal focus,
Escape, stacking, background inertness and timing remain browser-managed.
Visual parity and repeated first-attempt checks are required before acceptance.

## CI prerequisite

The `1369875` CI run skipped e2e because `golangci-lint` rejected the
protocol formatter's quote-state `if` chain (`gocritic`'s `ifElseChain`).
The formatter now uses a switch with the same branches. Its existing JSON
preservation tests cover the behavior; lint is checked alongside this slice.

## Online reload connection count

The supplied `40e5ec2` trace shows both players reconnecting before B reloads:
B has two previous WebSockets, then creates its third during reload. The
transport supports recovery from a dropped connection, but the spontaneous
drop in that trace remains unexplained; `FrameLog.sockets` counts every
connection ever created and never decreases when one closes. Waiting for an
old close cannot make that counter return to two.

The reload assertion now polls for exactly one connection above the count
immediately before reload, with a five-second bound. A focused case introduces
a prior offline/reconnect cycle, then checks the replacement count and the
same room version, seat and choices. A report annotation flags more than one connection before the intentional
reload without failing the test. Lifecycle attachments retain connection
opening/closing times and whether hello had a name, without identity tokens.
The transport behavior dates to `d0e08f8` (October 10); no networking code changes.

The direct icon renderer also omits groups that only apply an inherited fill
to leaf shapes, moving that fill onto each shape. Groups with transforms,
clipping, opacity or nesting remain. The art sources and shape coordinates
stay unchanged; original-paint parity and the existing DOM budget cover it.
One-unit resource costs put their existing tooltip and cost-pip marker on the
resource tile itself. Costs with a visible quantity keep their group and count.
This removes redundant HTML wrappers while retaining the hit target and labels.

## Runner performance remains open

The candidate's 10 first-attempt 4x runs per project pass locally, but CI run
39358029208 fails nine desktop and two mobile repetitions, with overlapping
tasks of 50 to 65 ms. Every failing task is in the `game end` room according
to the attached room records. Those records identify animation overlap, not
the native call within each task; a CI native trace would be needed for that.
This slice reduces style work and fixes bot presentation, but does not close
the M11 runner performance criterion. The follow-up ordering work must measure
both age transitions and game-end presentation.


## Completion drains and modal tasks

After the result barrier landed, runner CI still found an age-change task of
139 ms on desktop and 91 ms on mobile. The age opening drained only its deal:
coin counters continued for about 240 ms after its last card landed. The age
modal's native style recalculation could therefore block moving counters.
Age II/III opening frames now reuse the existing all-kind drain and its safety
limit, as final frames already do. Ordinary turns retain their short input hold.

A second boundary matters at game over. A source-mapped 6x trace captured a
55 ms task that ended victory 1.2 ms after entering, then spent about 40 ms in
`Sheet.showModal()`. Finishing the last animation resolved the hold in the same
task; the modal's reactive continuation still shared that task with completion.
An all-kind drain now reuses its hold timer for a zero-delay task handoff after
its last cleanup. The handoff checks that no new tail appeared. Interrupts and
room clearing still cancel the timer and release immediately. This lets the
completion task finish before native modal style/layout work starts, without
changing animation records, intersection rules or the 50 ms performance limit.

The age-order probe captures animations before native modal entry and checks
all opening effects, including counters and pulses. Unit tests check that the
handoff waits through cleanup microtasks, notices a late tail and is cancelled
by an interrupt. The former result-detail staging experiment was dropped: it
did not beat the barrier-only master in the local repeated comparison.
