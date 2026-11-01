import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "alma.auth.tokens";
let memoryTokens: AuthTokens | null = null;

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Stockage securise indisponible: la session reste en memoire.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    // Valeur corrompue ou stockage indisponible: on repart d'une session vide.
  }
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Rien a nettoyer sur les plateformes sans stockage securise.
  }
}
