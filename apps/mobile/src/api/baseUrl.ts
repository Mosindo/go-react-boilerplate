export const DEFAULT_API_PORT = 18080;

export function normalizeBaseUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}

/** EXPO_PUBLIC_API_URL wins; in dev fall back to the emulator-friendly host on the default port. */
export function resolveBaseUrl(input: {
  explicit: string | undefined;
  isDev: boolean;
  os: string;
}): string {
  const explicit = input.explicit?.trim();
  if (explicit) return normalizeBaseUrl(explicit);
  if (input.isDev) {
    const host = input.os === "android" ? "10.0.2.2" : "localhost";
    return `http://${host}:${DEFAULT_API_PORT}`;
  }
  return "";
}

export function toWebSocketUrl(baseUrl: string): string {
  return `${baseUrl.replace(/^http/i, "ws").replace(/\/+$/, "")}/ws`;
}
