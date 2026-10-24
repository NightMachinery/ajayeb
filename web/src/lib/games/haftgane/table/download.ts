// Saves a finished table's log exactly as GET /api/rooms/{code}/log answers
// it, under the server's filename (docs/protocol.md, "Downloading a game's
// log"). The endpoint decides whether the game is over and revealed. This
// repeats Duel's three lines ($lib/session/replay.ts) instead of importing
// them, so Haftgane's chunk adds no export to a module Duel loads.
export function downloadTableLog(code: string): void {
  const link = document.createElement('a');
  link.href = `/api/rooms/${encodeURIComponent(code)}/log`;
  link.download = '';
  document.body.append(link);
  link.click();
  link.remove();
}
