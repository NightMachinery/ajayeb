/** Saves exactly GET /api/rooms/{code}/log, with the server's filename.
 * The endpoint enforces finished/revealed access and supplies the UTC date. */
export function downloadReplay(code: string): void {
  const link = document.createElement('a');
  link.href = `/api/rooms/${encodeURIComponent(code)}/log`;
  link.download = '';
  document.body.append(link);
  link.click();
  link.remove();
}
