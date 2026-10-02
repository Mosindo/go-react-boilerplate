import Constants from "expo-constants";
import { Platform } from "react-native";

const API_PORT = 18080;

/**
 * Resolves the API base URL. Priority: EXPO_PUBLIC_API_URL, then the host of the Expo
 * dev server (so a physical phone reaches the computer's LAN IP, never "localhost"),
 * then localhost for web and emulators.
 */
export function resolveApiUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");

  const hostUri = Constants.expoConfig?.hostUri;
  if (hostUri) {
    const host = hostUri.split(":")[0];
    if (host) return `http://${host}:${API_PORT}`;
  }
  const loopback = Platform.OS === "android" ? "10.0.2.2" : "localhost";
  return `http://${loopback}:${API_PORT}`;
}

export const API_URL = resolveApiUrl();
