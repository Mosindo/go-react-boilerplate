import { getItem, removeItem, setItem } from "../storage";

export type Tokens = { accessToken: string; refreshToken: string };

const KEY = "braise.session.v1";
let current: Tokens | null = null;
const listeners = new Set<(tokens: Tokens | null) => void>();

function notify(): void {
  listeners.forEach((listener) => listener(current));
}

export function getTokens(): Tokens | null {
  return current;
}

export function onTokensChange(listener: (tokens: Tokens | null) => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export async function restoreTokens(): Promise<Tokens | null> {
  const raw = await getItem(KEY);
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as Partial<Tokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        current = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    } catch {
      current = null;
    }
  }
  notify();
  return current;
}

export async function storeTokens(tokens: Tokens): Promise<void> {
  current = tokens;
  await setItem(KEY, JSON.stringify(tokens));
  notify();
}

export async function clearTokens(): Promise<void> {
  current = null;
  await removeItem(KEY);
  notify();
}
