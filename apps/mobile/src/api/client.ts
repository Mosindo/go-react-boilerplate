import { Platform } from 'react-native';
import { resolveBaseUrl, resolveMediaUrl } from './baseUrl';
import { createHttpClient } from './http';
import { tokenStorage } from './tokenStorage';

export const API_BASE_URL = resolveBaseUrl({
  envUrl: process.env.EXPO_PUBLIC_API_URL,
  os: Platform.OS,
});

/** The single HTTP client of the app. UI code never calls fetch directly. */
export const http = createHttpClient({ baseUrl: API_BASE_URL, storage: tokenStorage });

/** Absolute URL for a (signed, relative) media URL returned by the API. */
export function mediaUrl(url: string): string {
  return resolveMediaUrl(API_BASE_URL, url);
}
