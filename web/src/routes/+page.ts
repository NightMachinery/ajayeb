// Only Home is rendered at build time. Game routes retain the client-only
// root layout and receive shell.html, never Home's hydration state.
export const ssr = true;
export const prerender = true;
