import { Platform } from "react-native";

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  // Development fallbacks only. Physical phones must set EXPO_PUBLIC_API_URL to the machine's LAN IP.
  if (Platform.OS === "android") return "http://10.0.2.2:18080";
  if (Platform.OS === "web" && typeof window !== "undefined")
    return `http://${window.location.hostname}:18080`;
  return "http://localhost:18080";
}

export const API_BASE_URL = resolveBaseUrl();

export function wsUrl(): string {
  return API_BASE_URL.replace(/^http/, "ws") + "/ws";
}

/** Photo URLs returned by the API are relative signed paths. */
export function mediaUrl(path: string | null | undefined): string | undefined {
  if (!path) return undefined;
  return path.startsWith("http") ? path : API_BASE_URL + path;
}
