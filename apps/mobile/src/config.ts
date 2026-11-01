import { Platform } from "react-native";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Emulateurs uniquement. Sur un telephone physique, definir EXPO_PUBLIC_API_URL
    // avec l'IP LAN de la machine (jamais localhost).
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();
export const WS_URL = API_BASE_URL.replace(/^http/, "ws") + "/ws";
