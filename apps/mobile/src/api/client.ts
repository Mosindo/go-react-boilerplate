import { Platform } from "react-native";
import { createAuthedFetch } from "../utils/authedFetch";
import { resolveAssetUrl, resolveBaseUrl } from "../utils/url";
import { tokenStore } from "./tokenStore";

export const API_BASE_URL = resolveBaseUrl({
  explicit: process.env.EXPO_PUBLIC_API_URL,
  isDev: __DEV__,
  os: Platform.OS,
});

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

let sessionLostHandler: () => void = () => undefined;
export function setSessionLostHandler(handler: () => void): void {
  sessionLostHandler = handler;
}

const authedFetch = createAuthedFetch({
  baseUrl: API_BASE_URL,
  fetchImpl: (input, init) => fetch(input, init),
  store: tokenStore,
  onSessionLost: () => sessionLostHandler(),
});

export function assetUrl(path: string): string {
  return resolveAssetUrl(API_BASE_URL, path);
}

type ErrorPayload = { error?: string; code?: string };

async function toApiError(res: Response): Promise<ApiError> {
  let payload: ErrorPayload = {};
  try {
    payload = (await res.json()) as ErrorPayload;
  } catch {
    // non-JSON error body
  }
  const fallback = res.status === 429 ? "Trop de requêtes, réessayez dans un instant." : "Une erreur est survenue.";
  return new ApiError(res.status, payload.code ?? "unknown", payload.error ?? fallback);
}

async function run(path: string, init: RequestInit): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 20_000);
  try {
    const res = await authedFetch(path, { ...init, signal: controller.signal });
    if (!res.ok) throw await toApiError(res);
    return res;
  } catch (err) {
    if (err instanceof ApiError) throw err;
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau et réessayez.");
  } finally {
    clearTimeout(timeout);
  }
}

export async function request<T>(path: string, options: { method?: string; body?: unknown } = {}): Promise<T> {
  const headers: Record<string, string> = {};
  let body: string | undefined;
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  const res = await run(path, { method: options.method ?? "GET", headers, body });
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Multipart upload of one image under the `photo` field. */
export async function uploadPhoto<T>(path: string, method: "POST" | "PUT", file: { uri: string; mimeType?: string }): Promise<T> {
  const form = new FormData();
  form.append("photo", {
    uri: file.uri,
    name: "photo.jpg",
    type: file.mimeType ?? "image/jpeg",
  } as unknown as Blob);
  const res = await run(path, { method, body: form });
  return (await res.json()) as T;
}

export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError;
}

export function errorMessage(err: unknown): string {
  if (isApiError(err)) return err.message;
  return "Une erreur est survenue.";
}
