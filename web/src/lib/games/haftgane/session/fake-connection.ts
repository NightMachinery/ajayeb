// A fake connection standing in for the entry's bootstrap in the Session's
// tests: it records what the Session sends and lets a test deliver frames,
// drop the link and rejoin.
import type { RoomListener } from '$lib/entry/bootstrap';
import type { TransportStatus } from '$lib/session/transport';
import type { RoomConnection } from './session.svelte';

export class FakeConnection implements RoomConnection {
  status: TransportStatus = { state: 'open', attempt: 0 };
  latest: unknown;
  listener: RoomListener | null = null;
  sent: Record<string, unknown>[] = [];
  failed: string[] = [];
  resyncs = 0;
  claims: (string | null)[] = [];
  attach(listener: RoomListener): unknown {
    this.listener = listener;
    const first = this.latest;
    this.latest = undefined;
    return first;
  }
  send(message: object): boolean {
    if (this.status.state !== 'open') return false;
    this.sent.push(JSON.parse(JSON.stringify(message)) as Record<string, unknown>);
    return true;
  }
  fail(code: string): void {
    this.failed.push(code);
  }
  resync(): void {
    this.resyncs++;
  }
  useClaim(claim: string | null): void {
    this.claims.push(claim);
  }
  frame(frame: unknown): void {
    this.listener!.message(frame);
  }
  drop(): void {
    this.status = { state: 'reconnecting', attempt: 1 };
    this.listener!.status(this.status);
  }
  /** A new socket welcomed: the bootstrap sent the join and says so. */
  rejoin(): void {
    this.status = { state: 'open', attempt: 0 };
    this.listener!.status(this.status);
    this.listener!.joined();
  }
  last(): Record<string, unknown> | undefined {
    return this.sent.at(-1);
  }
}
