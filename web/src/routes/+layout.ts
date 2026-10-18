// Routes are client-only by default. Home opts into build-time rendering in
// +page.ts; adapter-static writes shell.html for every other app path.
export const ssr = false;
export const prerender = false;
