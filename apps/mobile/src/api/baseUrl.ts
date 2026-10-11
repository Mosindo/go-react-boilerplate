export const DEFAULT_API_PORT = 18080;

export interface BaseUrlInput {
  /** Value of EXPO_PUBLIC_API_URL, if any. */
  envUrl: string | undefined;
  /** Platform.OS */
  os: string;
}

/** Resolves the API base URL: env override first, otherwise a dev fallback. */
export function resolveBaseUrl({ envUrl, os }: BaseUrlInput): string {
  const trimmed = envUrl?.trim();
  if (trimmed) {
    return trimmed.replace(/\/+$/, '');
  }
  const host = os === 'android' ? '10.0.2.2' : 'localhost';
  return `http://${host}:${DEFAULT_API_PORT}`;
}

/** Signed photo URLs are relative to the API base URL. */
export function resolveMediaUrl(baseUrl: string, url: string): string {
  if (/^[a-z][a-z0-9+.-]*:/i.test(url)) {
    return url;
  }
  return `${baseUrl}${url.startsWith('/') ? '' : '/'}${url}`;
}

/** http(s)://host/base -> ws(s)://host/base/ws?ticket=... */
export function toWsUrl(baseUrl: string, ticket: string): string {
  const wsBase = baseUrl.replace(/^http/i, (m) => (m.toLowerCase() === 'https' ? 'wss' : 'ws'));
  return `${wsBase}/ws?ticket=${encodeURIComponent(ticket)}`;
}
