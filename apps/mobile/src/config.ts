import { Platform } from "react-native";

function normalize(url: string): string {
  return url.replace(/\/+$/, "");
}

function resolveApiUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalize(explicit);
  }
  if (__DEV__) {
    // Emulator/simulator defaults. Physical devices MUST set EXPO_PUBLIC_API_URL to the LAN IP.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_URL = resolveApiUrl();
export const WS_URL = API_URL.replace(/^http/, "ws") + "/ws";
export const APP_NAME = "Tandem";
