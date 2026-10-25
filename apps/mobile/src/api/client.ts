import { API_BASE_URL } from "./baseUrl";
import { createHttpClient } from "./http";
import { tokenStore } from "./tokenStore";

const expiredListeners = new Set<() => void>();

/** Called when the refresh token is rejected: the user must sign in again. */
export function onSessionExpired(listener: () => void): () => void {
  expiredListeners.add(listener);
  return () => {
    expiredListeners.delete(listener);
  };
}

export const http = createHttpClient({
  baseUrl: API_BASE_URL,
  storage: tokenStore,
  onSessionExpired: () => {
    expiredListeners.forEach((listener) => listener());
  }
});
