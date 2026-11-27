import { readItem, removeItem, writeItem } from "../utils/storage";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

type Listener = (tokens: AuthTokens | null) => void;

const TOKENS_KEY = "lueur.auth.tokens";
let current: AuthTokens | null = null;
const listeners = new Set<Listener>();

function emit() {
  listeners.forEach((listener) => listener(current));
}

export function getTokens(): AuthTokens | null {
  return current;
}

export function subscribeTokens(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export async function restoreTokens(): Promise<AuthTokens | null> {
  const raw = await readItem(TOKENS_KEY);
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      current = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      return current;
    }
  } catch {
    // corrupted value: ignore and start signed out
  }
  await removeItem(TOKENS_KEY);
  return null;
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  current = tokens;
  emit();
  await writeItem(TOKENS_KEY, JSON.stringify(tokens));
}

export async function clearTokens(): Promise<void> {
  current = null;
  emit();
  await removeItem(TOKENS_KEY);
}
