import { readSecure, removeSecure, writeSecure } from "./storage";

export type Tokens = {
  accessToken: string;
  refreshToken: string;
};

const KEY = "lueur.session.v1";
type Listener = (tokens: Tokens | null) => void;

let current: Tokens | null = null;
const listeners = new Set<Listener>();

function emit() {
  listeners.forEach((listener) => listener(current));
}

export function getTokens(): Tokens | null {
  return current;
}

export function onTokensChange(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export async function restoreTokens(): Promise<Tokens | null> {
  const raw = await readSecure(KEY);
  if (!raw) {
    current = null;
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<Tokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      current = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      return current;
    }
  } catch {
    // fall through to cleanup
  }
  await removeSecure(KEY);
  current = null;
  return null;
}

export async function setTokens(tokens: Tokens): Promise<void> {
  current = tokens;
  await writeSecure(KEY, JSON.stringify(tokens));
  emit();
}

export async function clearTokens(): Promise<void> {
  current = null;
  await removeSecure(KEY);
  emit();
}
