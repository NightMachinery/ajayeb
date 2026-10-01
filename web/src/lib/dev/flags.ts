// Build flags for development-only features.

import * as publicEnv from '$env/static/public';

// A named import of PUBLIC_DEV_ROUTES would fail the build whenever the
// variable is unset. Reading it through a plain record compiles either way,
// and an unset variable means off.
const env: Readonly<Record<string, string | undefined>> = publicEnv;

/** The /dev routes (fixtures, specimens, gallery): always on under `vite dev`,
 *  and in a build only with PUBLIC_DEV_ROUTES=1 (see web/.env.example). */
export const devRoutesEnabled: boolean = import.meta.env.DEV || env['PUBLIC_DEV_ROUTES'] === '1';
