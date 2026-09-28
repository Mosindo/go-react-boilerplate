import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { API_BASE_URL } from "./config";
import { createApiClient } from "./client";

let sessionExpiredHandler: (() => void) | null = null;

/** The auth provider registers here so a failed refresh signs the user out everywhere. */
export function setSessionExpiredHandler(handler: (() => void) | null): void {
  sessionExpiredHandler = handler;
}

export const api = createApiClient({
  baseUrl: API_BASE_URL,
  fetchImpl: (input, init) => fetch(input, init),
  tokens: { get: getTokens, save: saveTokens, clear: clearTokens },
  onSessionExpired: () => sessionExpiredHandler?.()
});

export function getValidAccessToken(): Promise<string | null> {
  return api.getValidAccessToken();
}
