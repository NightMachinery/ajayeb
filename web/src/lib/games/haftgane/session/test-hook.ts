// window.__ajayebTable: a read-only window onto the table or lobby this
// page shows, for end-to-end drivers, beside Duel's window.__ajayeb
// ($lib/session/test-hook.ts). It exposes only what this tab already holds:
// the frames the server sent this viewer, never the identity token.

import type { LobbyState, TableHistoryEntry, TableState } from '$lib/wire/platform.gen';
import type { TablePresenter } from './presenter.svelte';
import type { ConnState, TableSession } from './session.svelte';

export interface TableTestHook {
  /** The table frame on screen, or null. */
  table(): TableState | null;
  /** The latest table frame the Session holds, or null. */
  latest(): TableState | null;
  /** The lobby frame, or null once the table started. */
  lobby(): LobbyState | null;
  conn(): ConnState;
  /** The request in flight: its family and seq. */
  pending(): { t: string; seq: number } | null;
  /** The last refusal or seat change shown. */
  notice(): { kind: string; code: string } | null;
  history(): readonly TableHistoryEntry[];
  /**
   * True when the page shows the latest frame and takes input: welcomed,
   * nothing queued in the Presenter, no request in flight.
   */
  ready(): boolean;
}

declare global {
  interface Window {
    __ajayebTable?: TableTestHook;
  }
}

/** Installs the hook for this session; the returned function removes it. */
export function installTableHook(session: TableSession, presenter: TablePresenter): () => void {
  if (typeof window === 'undefined') return () => {};
  const hook: TableTestHook = {
    table: () => presenter.frame,
    latest: () => session.table,
    lobby: () => session.lobby,
    conn: () => session.conn,
    pending: () => {
      const p = session.pending;
      return p ? { t: p.frame.t, seq: p.frame.seq } : null;
    },
    notice: () => {
      const n = session.notice;
      return n ? { kind: n.kind, code: n.code } : null;
    },
    history: () => session.history,
    ready: () =>
      session.conn === 'open' &&
      !session.pending &&
      (session.table ? presenter.idle && presenter.frame === session.table : session.lobby !== null)
  };
  window.__ajayebTable = hook;
  return () => {
    if (window.__ajayebTable === hook) delete window.__ajayebTable;
  };
}
