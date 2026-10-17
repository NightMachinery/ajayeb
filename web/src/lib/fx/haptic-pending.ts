// Pending device work shared with Settings without importing playback.
let waiting: ReturnType<typeof setTimeout> | null = null;

export function cancelHaptic(): void {
  if (waiting !== null) clearTimeout(waiting);
  waiting = null;
}

export function waitHaptic(play: () => void, delay: number): void {
  waiting = setTimeout(() => {
    waiting = null;
    play();
  }, delay);
}
