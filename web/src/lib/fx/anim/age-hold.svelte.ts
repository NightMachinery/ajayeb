// Whether a live frame's Age II or III deal holds the Presenter (plan.ts
// dealHold): set at that frame's capture, before its swap, and cleared when
// its last dealt card has landed or a new frame, hurry() or clear() ends
// it. It covers a deal carried past a skipped opening frame and the
// fallback deal on the first play frame of an age, not only the frame that
// opens on the starter choice. The game screen keeps the move summary and
// the pass-device screen off the board meanwhile (M11.age-start-order).

class AgeHold {
  /** An Age II or III deal holds the Presenter now. */
  active = $state(false);
}

/** The app's one age-deal hold. */
export const ageHold = new AgeHold();
