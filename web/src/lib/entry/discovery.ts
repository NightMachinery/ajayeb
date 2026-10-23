// Loaded only for a known non-Duel room or a welcomed v1 join refusal.
import type { Component } from 'svelte';
import { RoomBootstrap, snapshotKind } from './bootstrap';
import { loadEntry } from './registry';
import { entryIdentity, rememberEntryGame } from './identity';
import type { EntryProps } from './types';
import type { TransportStatus } from '$lib/session/transport';
import { savedRooms } from '$lib/session/saved-rooms.svelte';
interface Options {
  code: string;
  claim: string | null;
  ready: (
    entry: Component<EntryProps>,
    props: EntryProps,
    kind: 'duel' | 'haftgane' | 'lobby'
  ) => void;
  error: (code: string) => void;
  status: (status: TransportStatus) => void;
}
export class Discovery {
  #options: Options;
  #connection: RoomBootstrap;
  #disposed = false;
  #loading = false;
  #kind: ReturnType<typeof snapshotKind> = null;
  #module: Awaited<ReturnType<typeof loadEntry>> | undefined;
  constructor(options: Options) {
    this.#options = options;
    this.#connection = new RoomBootstrap({
      code: options.code,
      claim: options.claim,
      identity: entryIdentity(),
      snapshot: () => void this.#snapshot(),
      error: options.error,
      status: options.status
    });
  }
  async #snapshot(): Promise<void> {
    if (this.#disposed || this.#loading) return;
    // An overlapping Duel bootstrap is membership only. Its private stream is
    // never delivered to Session/Presenter while their v1 join is in flight.
    if (this.#kind === 'duel') return;
    const kind = snapshotKind(this.#connection.latest);
    if (!kind) return;
    this.#loading = true;
    try {
      const module = this.#kind === kind && this.#module ? this.#module : await loadEntry(kind);
      if (this.#disposed) return;
      if (snapshotKind(this.#connection.latest) !== kind) return;
      if (!module.accepts(this.#connection.latest, this.#options.code)) {
        this.#options.error('protocol');
        return;
      }
      this.#kind = kind;
      this.#module = module;
      rememberEntryGame(this.#options.code, kind);
      if (kind === 'haftgane')
        savedRooms().remember(
          this.#options.code,
          'online',
          Date.now(),
          this.#options.claim,
          'haftgane'
        );
      this.#options.ready(
        module.Component,
        kind === 'duel'
          ? {}
          : {
              connection: this.#connection,
              snapshot: this.#connection.latest,
              code: this.#options.code,
              claim: this.#options.claim
            },
        kind
      );
    } catch {
      if (!this.#disposed) this.#options.error('busy');
    } finally {
      this.#loading = false;
      if (!this.#disposed && snapshotKind(this.#connection.latest) !== kind) void this.#snapshot();
    }
  }
  /** The first ordinary Duel state proves the second membership is attached. */
  firstDuelState(code: string): void {
    if (this.#kind === 'duel' && code === this.#options.code) this.#connection.close();
  }
  close(): void {
    this.#disposed = true;
    this.#connection.close();
  }
}
