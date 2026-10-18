// Presentation memory only. Capture every authoritative boundary frame, even
// when the Presenter skips it during a fast bot reply or reconnect backlog.
// Views are immutable snapshots; retaining them freezes server totals without
// copying or recomputing game rules. The two ended ages bound this local queue.
import type { State, View } from '$lib/wire';

interface Boundary {
  view: View;
  starter: number | null;
  dismissed: boolean;
  dismissedSeats: readonly number[];
  shown: boolean;
  shownSeats: readonly number[];
}

export class AgeSummary {
  #boundaries = $state.raw<readonly Boundary[]>([]);
  #code: string | null = null;
  #ply = -1;

  // Keep the default shared-device accessors for standalone callers/tests.
  get view(): View | null {
    return this.viewFor();
  }
  get starter(): number | null {
    return this.starterFor();
  }
  get dismissed(): boolean {
    return this.isDismissed();
  }
  get shown(): boolean {
    return this.isShown();
  }

  #dismissed(boundary: Boundary, viewer: number | null): boolean {
    return viewer === null ? boundary.dismissed : boundary.dismissedSeats.includes(viewer);
  }
  #current(viewer: number | null): Boundary | undefined {
    return (
      this.#boundaries.find((boundary) => !this.#dismissed(boundary, viewer)) ??
      this.#boundaries.at(-1)
    );
  }
  viewFor(viewer: number | null = null): View | null {
    return this.#current(viewer)?.view ?? null;
  }
  starterFor(viewer: number | null = null): number | null {
    return this.#current(viewer)?.starter ?? null;
  }

  record(frame: State): void {
    const rewound = frame.code === this.#code && frame.view.ply < this.#ply;
    const boundary = frame.view.pending?.kind === 'chooseStarter';
    if (frame.code !== this.#code || frame.view.phase === 'wonderDraft') {
      this.#boundaries = [];
    } else {
      // Undo drops only boundaries no longer reached. Undoing a starter
      // decision resets that boundary, without erasing an older unread one.
      const retained = this.#boundaries.filter(
        (entry) =>
          frame.view.age >= entry.view.age &&
          frame.view.ply >= entry.view.ply &&
          !(rewound && boundary && frame.view.age === entry.view.age)
      );
      if (retained.length !== this.#boundaries.length) this.#boundaries = retained;
    }
    this.#code = frame.code;
    this.#ply = frame.view.ply;
    // A first starter-chosen frame is safe only before any card was taken.
    // Never relabel later scores, or an undo history resync, as ended totals.
    const chosen =
      !rewound &&
      !frame.history &&
      frame.view.phase === 'play' &&
      frame.view.structure.length > 0 &&
      frame.view.structure.every((slot) => slot.state !== 'empty') &&
      frame.events.some((event) => event.kind === 'starterChosen');
    if (
      (boundary || chosen) &&
      !this.#boundaries.some((entry) => entry.view.age === frame.view.age)
    ) {
      this.#boundaries = [
        ...this.#boundaries,
        {
          view: frame.view,
          starter: null,
          dismissed: false,
          dismissedSeats: [],
          shown: false,
          shownSeats: []
        }
      ].slice(-2);
    }
    const current = this.#boundaries.find((entry) => entry.view.age === frame.view.age);
    if (current && !frame.view.pending && current.starter === null) {
      const events = [...(frame.history?.flatMap((entry) => entry.events) ?? []), ...frame.events];
      const starter =
        events.findLast((event) => event.kind === 'starterChosen')?.starter ?? frame.view.active;
      this.#boundaries = this.#boundaries.map((entry) =>
        entry === current ? { ...entry, starter } : entry
      );
    }
  }

  #update(viewer: number | null, update: (boundary: Boundary) => Boundary): void {
    const current = this.#current(viewer);
    if (current)
      this.#boundaries = this.#boundaries.map((entry) =>
        entry === current ? update(entry) : entry
      );
  }
  markShown(viewer: number | null = null): void {
    this.#update(viewer, (entry) =>
      viewer === null
        ? { ...entry, shown: true }
        : entry.shownSeats.includes(viewer)
          ? entry
          : { ...entry, shownSeats: [...entry.shownSeats, viewer] }
    );
  }
  isShown(viewer: number | null = null): boolean {
    const current = this.#current(viewer);
    return !!current && (viewer === null ? current.shown : current.shownSeats.includes(viewer));
  }
  isDismissed(viewer: number | null = null): boolean {
    const current = this.#current(viewer);
    return !!current && this.#dismissed(current, viewer);
  }
  close(viewer: number | null = null): void {
    this.#update(viewer, (entry) =>
      viewer === null
        ? { ...entry, dismissed: true }
        : entry.dismissedSeats.includes(viewer)
          ? entry
          : { ...entry, dismissedSeats: [...entry.dismissedSeats, viewer] }
    );
  }
  reopen(viewer: number | null = null): void {
    this.#update(viewer, (entry) =>
      viewer === null
        ? { ...entry, dismissed: false }
        : { ...entry, dismissedSeats: entry.dismissedSeats.filter((seat) => seat !== viewer) }
    );
  }
}
