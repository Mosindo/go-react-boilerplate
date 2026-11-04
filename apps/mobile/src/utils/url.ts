export function normalizeBaseUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}

export type BaseUrlInput = { explicit?: string; isDev: boolean; os: string };

/** Resolves the API base URL. Empty string means "not configured". */
export function resolveBaseUrl({ explicit, isDev, os }: BaseUrlInput): string {
  if (explicit && explicit.trim()) return normalizeBaseUrl(explicit);
  if (isDev) return os === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  return "";
}

/** Turns an API-relative path (signed photo URL) into an absolute URL. */
export function resolveAssetUrl(base: string, path: string): string {
  if (/^https?:\/\//i.test(path)) return path;
  return `${base}${path.startsWith("/") ? "" : "/"}${path}`;
}

/** http(s) -> ws(s) endpoint for the realtime socket. */
export function websocketUrl(base: string): string {
  return `${base.replace(/^http/i, "ws")}/ws`;
}
