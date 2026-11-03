import { Platform } from "react-native";

function normalize(value: string): string {
  return value.replace(/\/+$/, "");
}

/**
 * API base URL. Physical devices need EXPO_PUBLIC_API_URL set to the LAN IP of
 * the machine running the API (never localhost). Emulators/web fall back to the
 * conventional loopback addresses of the Docker-exposed port 18080.
 */
function resolveApiUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalize(explicit);
  }
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

export const API_URL = resolveApiUrl();
export const WS_URL = `${API_URL.replace(/^http/, "ws")}/ws`;
