// Utility to expose the user's color-scheme preference so it can be reused
// across the codebase. We export an initial boolean `isDark`, a runtime
// accessor `isDarkNow()` and a helper to register listeners for changes.

export const isDark =
  typeof globalThis.matchMedia === "function"
    ? globalThis.matchMedia("(prefers-color-scheme: dark)").matches
    : false;

export function isDarkNow() {
  if (typeof globalThis.matchMedia === "function") {
    return globalThis.matchMedia("(prefers-color-scheme: dark)").matches;
  }
  return false;
}

// Adds a listener for changes to the prefers-color-scheme media query.
// The callback is called with the new boolean value (true when dark).
// Returns a cleanup function to remove the listener.
export function addDarkModeListener(cb) {
  if (typeof globalThis.matchMedia !== "function") {
    return () => {};
  }
  const mq = globalThis.matchMedia("(prefers-color-scheme: dark)");
  const handler = (e) =>
    cb(!!(e && typeof e.matches === "boolean" ? e.matches : mq.matches));
  // Use addEventListener if available; matchMedia listeners use the same API in modern browsers
  mq.addEventListener("change", handler);
  return () => mq.removeEventListener("change", handler);
}
