import { Platform } from 'react-native';

function normalize(value: string): string {
  return value.trim().replace(/\/+$/, '');
}

export function resolveApiBaseUrl(
  explicit: string | undefined,
  os: string,
  isDev: boolean,
): string {
  if (explicit && explicit.trim()) return normalize(explicit);
  if (isDev) return os === 'android' ? 'http://10.0.2.2:18080' : 'http://localhost:18080';
  return '';
}

export const API_BASE_URL = resolveApiBaseUrl(
  process.env.EXPO_PUBLIC_API_URL,
  Platform.OS,
  typeof __DEV__ !== 'undefined' ? __DEV__ : false,
);

/** http(s)://host -> ws(s)://host */
export function toWebSocketUrl(httpUrl: string): string {
  return httpUrl.replace(/^http/i, 'ws');
}
