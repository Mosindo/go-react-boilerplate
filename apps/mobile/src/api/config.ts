import { Platform } from "react-native";
import { resolveBaseUrl } from "./baseUrl";

export const API_BASE_URL = resolveBaseUrl({
  explicit: process.env.EXPO_PUBLIC_API_URL,
  isDev: __DEV__,
  os: Platform.OS
});
