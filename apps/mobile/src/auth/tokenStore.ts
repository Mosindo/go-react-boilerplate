import * as SecureStore from "expo-secure-store";
import type { Tokens } from "../api/client";

const KEY = "ember.session";

export async function loadTokens(): Promise<Tokens | null> {
  try {
    const raw = await SecureStore.getItemAsync(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<Tokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // unreadable or unavailable secure storage: behave as signed out
  }
  return null;
}

export async function persistTokens(tokens: Tokens | null): Promise<void> {
  try {
    if (tokens) await SecureStore.setItemAsync(KEY, JSON.stringify(tokens));
    else await SecureStore.deleteItemAsync(KEY);
  } catch {
    // the in-memory session still works for this launch
  }
}
