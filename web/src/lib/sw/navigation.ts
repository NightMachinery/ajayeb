/** Network-first HTML loading. An HTTP error is still a network answer:
 *  use the offline shell only for a failed or slow connection. */
export async function networkFirst(
  fetchNetwork: () => Promise<Response>,
  loadShell: () => Promise<Response | undefined>,
  timeoutMs: number
): Promise<Response> {
  const network = Promise.resolve().then(fetchNetwork);
  // The timeout may win before the network rejects.
  void network.catch(() => undefined);
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<undefined>((resolve) => {
    timer = setTimeout(() => resolve(undefined), timeoutMs);
  });
  try {
    const response = await Promise.race([network, timeout]);
    if (response) return response;
  } catch {
    // Offline or refused: use the cached shell below.
  } finally {
    clearTimeout(timer);
  }
  return (await loadShell()) ?? network;
}
