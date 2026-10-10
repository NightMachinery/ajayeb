// A small toast queue as a plain runes class, no store library needed (see
// AGENTS.md: Svelte 5 runes only). One instance lives for the life of the
// app (a context value once routes need it); pair it with Toasts.svelte for
// the visible list. Named toast-queue.svelte.ts, not toasts.svelte.ts,
// because that name differs from Toasts.svelte only in case; see the note
// in Toasts.svelte.
//
//   const toasts = new Toasts();
//   toasts.show(t('board.built', { card: name }), { kind: 'success' });

export type ToastKind = 'info' | 'success' | 'warning' | 'danger';

export interface ToastOptions {
  kind?: ToastKind;
  /** Auto-dismiss after this many ms; 0 keeps it until dismissed by hand. */
  duration?: number;
  actionLabel?: string;
  onAction?: () => void;
}

export interface ToastItem {
  readonly id: number;
  readonly message: string;
  readonly kind: ToastKind;
  readonly actionLabel?: string;
  readonly onAction?: () => void;
}

const DEFAULT_DURATION_MS = 5000;

export class Toasts {
  items = $state<ToastItem[]>([]);
  #nextId = 1;
  // Plain Map on purpose: timer handles are never read reactively (nothing
  // renders from this), only looked up by id to clear(), so a SvelteMap's
  // extra tracking would be pure overhead.
  // eslint-disable-next-line svelte/prefer-svelte-reactivity
  #timers = new Map<number, ReturnType<typeof setTimeout>>();

  /** Queues a toast and returns its id, so the caller can dismiss() it early. */
  show(message: string, options: ToastOptions = {}): number {
    const id = this.#nextId++;
    const { kind = 'info', duration = DEFAULT_DURATION_MS, actionLabel, onAction } = options;
    this.items.push({ id, message, kind, actionLabel, onAction });
    if (duration > 0) {
      this.#timers.set(
        id,
        setTimeout(() => this.dismiss(id), duration)
      );
    }
    return id;
  }

  dismiss(id: number): void {
    const timer = this.#timers.get(id);
    if (timer !== undefined) {
      clearTimeout(timer);
      this.#timers.delete(id);
    }
    this.items = this.items.filter((item) => item.id !== id);
  }

  /** Clears every toast and cancels their timers (route change, teardown). */
  clear(): void {
    for (const timer of this.#timers.values()) clearTimeout(timer);
    this.#timers.clear();
    this.items = [];
  }
}
