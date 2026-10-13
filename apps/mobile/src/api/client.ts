import { Platform } from "react-native";
import { showGlobalError } from "../shared/feedback/store";
import { clearTokens, loadTokens, saveTokens } from "../store/tokenStore";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Android emulators reach the host through 10.0.2.2. Physical phones need
    // EXPO_PUBLIC_API_URL=http://<LAN_IP>:<PORT>.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export class ApiError extends Error {
  public status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  json?: unknown;
  form?: FormData;
  signal?: AbortSignal;
  /** Skip the Authorization header and the refresh-on-401 behaviour. */
  anonymous?: boolean;
  timeoutMs?: number;
};

function networkHint(): string {
  if (!API_BASE_URL) {
    return "L'adresse de l'API n'est pas configurée. Définissez EXPO_PUBLIC_API_URL.";
  }
  if (Platform.OS === "android") {
    return "Connexion impossible. Sur un téléphone, utilisez l'IP du réseau local dans EXPO_PUBLIC_API_URL.";
  }
  return "Connexion impossible. Vérifiez votre réseau puis réessayez.";
}

let refreshInFlight: Promise<boolean> | null = null;

/** Rotates the refresh token once even when several requests fail together. */
function refreshSession(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await loadTokens();
      if (!tokens?.refreshToken) {
        return false;
      }
      try {
        const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken })
        });
        if (!response.ok) {
          // Only a definitive rejection ends the session; transient errors keep it.
          if (response.status === 401 || response.status === 403) {
            await clearTokens();
          }
          return false;
        }
        const body = (await response.json()) as { accessToken: string; refreshToken: string };
        await saveTokens({ accessToken: body.accessToken, refreshToken: body.refreshToken });
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

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.json !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.json);
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } finally {
    clearTimeout(timeout);
  }
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  if (!API_BASE_URL) {
    showGlobalError(networkHint());
    throw new ApiError(0, networkHint());
  }

  let response: Response;
  try {
    const tokens = options.anonymous ? null : await loadTokens();
    response = await send(path, options, tokens?.accessToken ?? null);
    if (response.status === 401 && !options.anonymous && tokens) {
      if (await refreshSession()) {
        const renewed = await loadTokens();
        response = await send(path, options, renewed?.accessToken ?? null);
      }
    }
  } catch (error) {
    const aborted = (error as { name?: string }).name === "AbortError";
    const message = aborted ? "La requête a expiré. Réessayez." : networkHint();
    showGlobalError(message);
    throw new ApiError(0, message);
  }

  if (!response.ok) {
    let message = `Erreur ${response.status}`;
    try {
      const data = (await response.json()) as { error?: string };
      if (data.error) {
        message = data.error;
      }
    } catch {
      // keep the generic message
    }
    if (response.status >= 500) {
      showGlobalError("Le serveur rencontre un problème. Réessayez dans un instant.");
    }
    throw new ApiError(response.status, message);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function isApiError(error: unknown, status?: number): error is ApiError {
  return error instanceof ApiError && (status === undefined || error.status === status);
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
