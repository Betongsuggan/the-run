// Admin section is client-side only: it depends on browser-only state
// (localStorage auth + query params). Rely on the static fallback instead.
export const prerender = false;
export const ssr = false;
