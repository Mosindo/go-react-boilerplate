export function formatDistance(km: number | undefined): string | null {
  if (km === undefined || km === null) {
    return null;
  }
  return km <= 1 ? "less than 1 km away" : `${km} km away`;
}

/** Turns an http(s) API base URL into the matching WebSocket endpoint. */
export function toWebSocketUrl(baseUrl: string): string {
  return `${baseUrl.replace(/^http/i, "ws").replace(/\/+$/, "")}/ws`;
}

export function validateEmail(value: string): string | null {
  const email = value.trim();
  if (!email) {
    return "Enter your email address.";
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return "That email address does not look right.";
  }
  return null;
}

export function validatePassword(value: string): string | null {
  const bytes = new TextEncoder().encode(value).length;
  if (bytes < 8) {
    return "Use at least 8 characters.";
  }
  if (bytes > 72) {
    return "Use at most 72 characters.";
  }
  return null;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

export const GENDER_LABELS = { man: "Man", woman: "Woman", nonbinary: "Non-binary" } as const;
