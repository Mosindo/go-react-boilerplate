import { Platform } from "react-native";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Emulator/simulator only. Physical phones must set EXPO_PUBLIC_API_URL to the computer's LAN IP.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export function toWebSocketUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/^http/, "ws")}${path}`;
}
