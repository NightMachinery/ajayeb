# Master e2e failure attribution

## Persian Continue age label

`65d3937` (M11.wording-consistency, October 18) changed Home to use the
shared `ageLabel` helper. In Persian, Age II is written as `دوران دوم`.
The returning-device hydration test still expected the digit `۲`, so the
test was stale. It now checks the actual localized age wording while
retaining its hydration, theme, tutorial-offer and saved-room assertions.
The comparison ignores invisible bidi isolation marks, as the wording
consistency tests already do.
