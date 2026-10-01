// A single-page app: nothing renders on a server and nothing is prerendered.
// adapter-static writes index.html as the fallback for every path
// (svelte.config.js), and the client router renders the route.
export const ssr = false;
export const prerender = false;
