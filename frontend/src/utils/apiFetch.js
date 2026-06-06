export function apiFetch(input, init = {}) {
  const headers = new Headers(init.headers || {});
  if (!headers.has("X-DockLog-Client")) {
    headers.set("X-DockLog-Client", "web");
  }
  return fetch(input, { ...init, headers });
}
