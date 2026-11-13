import { Platform } from "react-native";
import { getTokens, setTokens, type Tokens } from "./session";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Emulators: Android reaches the host through 10.0.2.2. Physical phones
    // need EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();
const REQUEST_TIMEOUT_MS = 15_000;

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export class NetworkError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "NetworkError";
  }
}

/** Absolute URL for API-relative paths such as signed media URLs. */
export function apiUrl(path: string): string {
  return path.startsWith("http") ? path : `${API_BASE_URL}${path}`;
}

let refreshInFlight: Promise<Tokens | null> | null = null;
let onSessionExpired: (() => void) | null = null;

export function setSessionExpiredHandler(handler: (() => void) | null) {
  onSessionExpired = handler;
}

/** Single-flight refresh shared by every concurrent request. */
export function refreshTokens(): Promise<Tokens | null> {
  if (refreshInFlight) {
    return refreshInFlight;
  }
  const tokens = getTokens();
  if (!tokens?.refreshToken) {
    return Promise.resolve(null);
  }
  refreshInFlight = (async () => {
    try {
      const response = await fetchWithTimeout(apiUrl("/auth/refresh"), {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ refreshToken: tokens.refreshToken })
      });
      if (!response.ok) {
        if (response.status === 401) {
          await setTokens(null);
          onSessionExpired?.();
        }
        return null;
      }
      const data = (await response.json()) as { accessToken: string; refreshToken: string };
      const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
      await setTokens(next);
      return next;
    } catch {
      return null;
    } finally {
      refreshInFlight = null;
    }
  })();
  return refreshInFlight;
}

async function fetchWithTimeout(url: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    return await fetch(url, { ...init, signal: controller.signal });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new NetworkError("La connexion est trop lente. Réessayez.");
    }
    throw new NetworkError(
      API_BASE_URL
        ? "Impossible de joindre le serveur. Vérifiez votre connexion."
        : "EXPO_PUBLIC_API_URL n'est pas configurée pour cette version de l'app."
    );
  } finally {
    clearTimeout(timer);
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  formData?: FormData;
  auth?: boolean;
};

async function parseError(response: Response): Promise<ApiError> {
  let message = `Erreur ${response.status}`;
  let code: string | undefined;
  try {
    const data = (await response.json()) as { error?: string; code?: string };
    if (data.error) {
      message = data.error;
    }
    code = data.code;
  } catch {
    // Non-JSON error body.
  }
  return new ApiError(response.status, message, code);
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, formData, auth = true } = options;

  const send = async (accessToken: string | undefined) => {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
    }
    if (auth && accessToken) {
      headers.Authorization = `Bearer ${accessToken}`;
    }
    return fetchWithTimeout(apiUrl(path), {
      method,
      headers,
      body: formData ?? (body !== undefined ? JSON.stringify(body) : undefined)
    });
  };

  let response = await send(getTokens()?.accessToken);
  if (response.status === 401 && auth && getTokens()?.refreshToken) {
    const refreshed = await refreshTokens();
    if (refreshed) {
      response = await send(refreshed.accessToken);
    }
  }
  if (!response.ok) {
    throw await parseError(response);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
