import { Platform } from "react-native";
import { API_BASE_URL } from "../config";
import { clearTokens, currentTokens, saveTokens } from "../store/tokenStore";

export class ApiError extends Error {
  readonly status: number;
  readonly field?: string;

  constructor(status: number, message: string, field?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.field = field;
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  /** Skip the Authorization header (login, register, ...). */
  anonymous?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

type SessionExpiredHandler = () => void;
let onSessionExpired: SessionExpiredHandler = () => undefined;

export function setSessionExpiredHandler(handler: SessionExpiredHandler): void {
  onSessionExpired = handler;
}

function networkMessage(): string {
  if (!API_BASE_URL) {
    return "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).";
  }
  if (__DEV__ && Platform.OS !== "web") {
    return "Impossible de joindre le serveur. Sur un téléphone, utilisez l'IP de votre ordinateur dans EXPO_PUBLIC_API_URL (pas localhost).";
  }
  return "Impossible de joindre le serveur. Vérifiez votre connexion.";
}

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (token && !options.anonymous) {
    headers.Authorization = `Bearer ${token}`;
  }
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if (options.signal?.aborted) {
      throw error;
    }
    const timedOut = (error as { name?: string }).name === "AbortError";
    throw new ApiError(0, timedOut ? "Le serveur met trop de temps à répondre." : networkMessage());
  } finally {
    clearTimeout(timeout);
  }
}

let refreshInFlight: Promise<boolean> | null = null;

/** Exchanges the refresh token once, even when several requests fail together. */
function refreshTokens(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = currentTokens();
      if (!tokens?.refreshToken) {
        return false;
      }
      try {
        const response = await send(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: tokens.refreshToken }, anonymous: true },
          null
        );
        if (!response.ok) {
          return false;
        }
        const data = (await response.json()) as { accessToken: string; refreshToken: string };
        await saveTokens({ accessToken: data.accessToken, refreshToken: data.refreshToken });
        return true;
      } catch {
        return false;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function toApiError(response: Response): Promise<ApiError> {
  let message = `Erreur ${response.status}`;
  let field: string | undefined;
  try {
    const data = (await response.json()) as { error?: string; field?: string };
    if (data.error) {
      message = data.error;
    }
    field = data.field;
  } catch {
    // Non-JSON error body.
  }
  if (response.status === 429) {
    message = "Trop de tentatives. Réessayez dans quelques instants.";
  }
  return new ApiError(response.status, message, field);
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  let response = await send(path, options, currentTokens()?.accessToken ?? null);

  if (response.status === 401 && !options.anonymous && currentTokens()) {
    if (await refreshTokens()) {
      response = await send(path, options, currentTokens()?.accessToken ?? null);
    }
    if (response.status === 401) {
      await clearTokens();
      onSessionExpired();
    }
  }

  if (!response.ok) {
    throw await toApiError(response);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
