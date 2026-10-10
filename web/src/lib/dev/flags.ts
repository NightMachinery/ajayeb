// Build flags for development-only features.

import * as publicEnv from '$env/static/public';

// A named import of PUBLIC_DEV_ROUTES would fail the build whenever the
// variable is unset. Reading it through a plain record compiles either way,
// and an unset variable means off.
const env: Readonly<Record<string, string | undefined>> = publicEnv;

/** The /dev routes (fixtures, specimens, gallery): always on under `vite dev`
 *  and in the test-mode build (`vite build --mode test`, which end-to-end tests
 *  and screenshots use), and in any other build only with PUBLIC_DEV_ROUTES=1
 *  (see web/.env.example). The production build leaves them off. */
export const devRoutesEnabled: boolean =
  import.meta.env.DEV || import.meta.env.MODE === 'test' || env['PUBLIC_DEV_ROUTES'] === '1';
