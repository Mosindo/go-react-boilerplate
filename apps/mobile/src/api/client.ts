import { Platform } from "react-native";
import { getTokens } from "../store/tokenStore";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Emulator defaults only. Physical phones must set EXPO_PUBLIC_API_URL to the LAN IP.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();
const REQUEST_TIMEOUT_MS = 15_000;

export class ApiError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export class NetworkError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "NetworkError";
  }
}

type RefreshHandler = () => Promise<string | null>;
let refreshHandler: RefreshHandler | null = null;
let refreshInFlight: Promise<string | null> | null = null;

/** The auth layer registers how to obtain a fresh access token after a 401. */
export function setRefreshHandler(handler: RefreshHandler | null) {
  refreshHandler = handler;
}

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshHandler) {
    return null;
  }
  // Single flight: concurrent 401s share one refresh call (refresh tokens rotate).
  refreshInFlight ??= refreshHandler().finally(() => {
    refreshInFlight = null;
  });
  return refreshInFlight;
}

const FRIENDLY: Record<string, string> = {
  "invalid credentials": "Email ou mot de passe incorrect.",
  "email already exists": "Un compte existe déjà avec cet email.",
  "invalid email address": "Adresse email invalide.",
  "password must be 8-128 characters and contain a letter and a digit": "Le mot de passe doit contenir 8 caractères minimum, dont une lettre et un chiffre.",
  "invalid or expired reset code": "Code invalide ou expiré.",
  "you must be at least 18 years old": "Vous devez avoir au moins 18 ans.",
  "too many requests, please slow down": "Trop de tentatives, réessayez dans un instant.",
  "only JPEG, PNG or WebP images are accepted": "Formats acceptés : JPEG, PNG ou WebP.",
  "photo is too large (10 MB max)": "Photo trop lourde (10 Mo max).",
  "image is too small (200px minimum)": "Image trop petite (200 px minimum).",
  "you can upload up to 6 photos": "6 photos maximum.",
  "conversation not found": "Cette conversation n'est plus disponible.",
  "this profile is no longer available": "Ce profil n'est plus disponible.",
  "you already responded to this profile": "Vous avez déjà répondu à ce profil.",
  "current password is incorrect": "Mot de passe actuel incorrect.",
  "password is incorrect": "Mot de passe incorrect.",
  "birthdate cannot be changed once set": "La date de naissance ne peut plus être modifiée.",
  "first name must be 1-40 letters": "Le prénom doit contenir uniquement des lettres (40 max)."
};

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof NetworkError) {
    return error.message;
  }
  if (error instanceof ApiError) {
    if (error.status >= 500) {
      return "Le service est momentanément indisponible. Réessayez.";
    }
    return FRIENDLY[error.message] ?? error.message ?? fallback;
  }
  return fallback;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
};

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new NetworkError("L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) {
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
    return await fetch(`${API_BASE_URL}${path}`, { method: options.method ?? "GET", headers, body, signal: controller.signal });
  } catch (error) {
    if (options.signal?.aborted) {
      throw error;
    }
    const name = (error as { name?: string }).name;
    throw new NetworkError(
      name === "AbortError" ? "La requête a expiré. Vérifiez votre connexion." : "Impossible de joindre le serveur. Vérifiez votre connexion."
    );
  } finally {
    clearTimeout(timeout);
  }
}

/** The only way UI code talks to the network. */
export async function api<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let token = useAuth ? ((await getTokens())?.accessToken ?? null) : null;
  let response = await send(path, options, token);

  if (response.status === 401 && useAuth && token) {
    token = await refreshAccessToken();
    if (token) {
      response = await send(path, options, token);
    }
  }

  const text = await response.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }
  if (!response.ok) {
    const message = (data as { error?: string } | null)?.error ?? `Erreur ${response.status}`;
    throw new ApiError(response.status, message);
  }
  return data as T;
}
