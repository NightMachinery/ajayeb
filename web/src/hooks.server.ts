import { building } from '$app/environment';
import { prioritizeHomePaint } from '$lib/home/head';
import type { Handle } from '@sveltejs/kit';

// This runs while building static Home, never in the Go server. Keep the
// generic client shell and all other routes exactly as SvelteKit emits them.
export const handle: Handle = ({ event, resolve }) => {
  if (!building || event.route.id !== '/') return resolve(event);
  return resolve(event, {
    transformPageChunk: ({ html }) => prioritizeHomePaint(html)
  });
};
