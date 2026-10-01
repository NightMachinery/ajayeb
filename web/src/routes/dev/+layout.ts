// Every /dev route (fixtures, specimens, gallery) answers 404 unless dev
// routes are enabled: always under `vite dev`, and in a build only with
// PUBLIC_DEV_ROUTES=1 (src/lib/dev/flags.ts).
import { error } from '@sveltejs/kit';
import { devRoutesEnabled } from '$lib/dev/flags';
import type { LayoutLoad } from './$types';

export const prerender = false;

export const load: LayoutLoad = () => {
  if (!devRoutesEnabled) error(404, 'Not Found');
};
