import { Platform } from "react-native";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function developmentFallbackBaseUrl(): string {
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

function explicitApiUrl(): string {
  const value: unknown = process.env.EXPO_PUBLIC_API_URL;
  return typeof value === "string" ? value.trim() : "";
}

function resolveApiBaseUrl(): string {
  const explicitBaseUrl = explicitApiUrl();
  if (explicitBaseUrl) {
    return normalizeBaseUrl(explicitBaseUrl);
  }
  return __DEV__ ? developmentFallbackBaseUrl() : "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export function hasExplicitApiUrl(): boolean {
  return explicitApiUrl() !== "";
}
