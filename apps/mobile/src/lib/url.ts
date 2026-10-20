export function normalizeBaseUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}

export function toWebSocketUrl(apiBaseUrl: string): string {
  const base = normalizeBaseUrl(apiBaseUrl);
  if (base.startsWith("https://")) return `wss://${base.slice("https://".length)}/ws`;
  if (base.startsWith("http://")) return `ws://${base.slice("http://".length)}/ws`;
  return `${base}/ws`;
}

export function photoUri(apiBaseUrl: string, relativeUrl: string): string {
  return `${normalizeBaseUrl(apiBaseUrl)}${relativeUrl.startsWith("/") ? "" : "/"}${relativeUrl}`;
}
