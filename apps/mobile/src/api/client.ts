import { tokenStorage } from "../store/tokenStore";
import { API_BASE_URL } from "./config";
import { createHttpClient } from "./http";

type Listener = () => void;
const expiredListeners = new Set<Listener>();

/** Subscribe to "refresh token rejected" (the session provider signs out). */
export function onSessionExpired(listener: Listener): () => void {
  expiredListeners.add(listener);
  return () => expiredListeners.delete(listener);
}

export const http = createHttpClient({
  baseUrl: API_BASE_URL,
  storage: tokenStorage,
  onSessionExpired: () => expiredListeners.forEach((listener) => listener())
});

export const request = http.request;
