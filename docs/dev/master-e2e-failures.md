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
live move's events. A cosmetic update to the displayed position updates
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
clipping ids, removing the sprite and SVG use shadow trees. The age-summary
diagnostic restyles 1,870 elements instead of 2,632 on desktop, and 1,183
instead of 1,747 on mobile. The fixed-scale pixel comparison checks all
81 icons in both themes and with inherited two-tone paints. Native modal focus,
Escape, stacking, background inertness and timing remain browser-managed.
Visual parity and repeated first-attempt checks are required before acceptance.

## CI prerequisite

The `1369875` CI run skipped e2e because `golangci-lint` rejected the
protocol formatter's quote-state `if` chain (`gocritic`'s `ifElseChain`).
The formatter now uses a switch with the same branches. Its existing JSON
preservation tests cover the behavior; lint is checked alongside this slice.
