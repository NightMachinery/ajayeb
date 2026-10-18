// Only personal turn notifications wait on a cancellable device timer.
// Ordinary event sounds keep their existing WebAudio scheduling.
let waiting: ReturnType<typeof setTimeout> | null = null;

export function cancelTurnCue(): void {
  if (waiting !== null) clearTimeout(waiting);
  waiting = null;
}

export function waitTurnCue(play: () => void, delay: number): void {
  cancelTurnCue();
  waiting = setTimeout(
    () => {
      waiting = null;
      play();
    },
    Math.max(0, delay)
  );
}
