/** Returns the `exp` claim (seconds since epoch) of a JWT, or null if unreadable. */
export function tokenExpiry(token: string): number | null {
  const part = token.split(".")[1];
  if (!part) {
    return null;
  }
  try {
    const base64 = part.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
    const payload = JSON.parse(atob(padded)) as { exp?: unknown };
    return typeof payload.exp === "number" ? payload.exp : null;
  } catch {
    return null;
  }
}

/** True when the token is missing an expiry or expires within `skewSeconds`. */
export function isExpiring(token: string, nowMs: number, skewSeconds = 30): boolean {
  const exp = tokenExpiry(token);
  return exp === null || exp * 1000 - nowMs < skewSeconds * 1000;
}
