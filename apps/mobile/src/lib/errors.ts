import { strings } from "./strings";

type ErrorLike = {
  status?: unknown;
  message?: unknown;
  name?: unknown;
  retryAfterSeconds?: unknown;
};

function asErrorLike(error: unknown): ErrorLike | null {
  return typeof error === "object" && error !== null ? (error as ErrorLike) : null;
}

export function errorStatus(error: unknown): number | null {
  const status = asErrorLike(error)?.status;
  return typeof status === "number" ? status : null;
}

/**
 * Maps any thrown value to a message that is safe and friendly to show.
 * Server messages (docs/API.md: always human readable) are used for 4xx, but never for 5xx.
 */
export function messageFromError(error: unknown, fallback: string = strings.errors.generic): string {
  const like = asErrorLike(error);
  if (!like) {
    return fallback;
  }
  const status = errorStatus(error);
  const message = typeof like.message === "string" ? like.message.trim() : "";

  if (status === null) {
    if (like.name === "TypeError") {
      return strings.errors.network;
    }
    if (like.name === "TimeoutError" || like.name === "AbortError") {
      return strings.errors.timeout;
    }
    return message || fallback;
  }
  if (status === 429) {
    const seconds = typeof like.retryAfterSeconds === "number" ? Math.ceil(like.retryAfterSeconds) : null;
    return seconds && seconds > 0
      ? `Too many attempts. Please wait ${seconds} second${seconds === 1 ? "" : "s"} and try again.`
      : strings.errors.rateLimited;
  }
  if (status >= 500) {
    return strings.errors.server;
  }
  if (status === 413) {
    return strings.errors.tooLarge;
  }
  if (status === 401) {
    return message || strings.errors.unauthorized;
  }
  if (message && !/^API request failed/.test(message)) {
    return message;
  }
  if (status === 403) {
    return strings.errors.forbidden;
  }
  if (status === 404) {
    return strings.errors.notFound;
  }
  return fallback;
}

export function isStatus(error: unknown, ...statuses: number[]): boolean {
  const status = errorStatus(error);
  return status !== null && statuses.includes(status);
}

/** Parses a Retry-After header (delta-seconds or HTTP date). Returns whole seconds or null. */
export function parseRetryAfter(value: string | null | undefined, now: number = Date.now()): number | null {
  if (!value) {
    return null;
  }
  const trimmed = value.trim();
  if (/^\d+$/.test(trimmed)) {
    return Number(trimmed);
  }
  const date = Date.parse(trimmed);
  if (Number.isNaN(date)) {
    return null;
  }
  return Math.max(0, Math.ceil((date - now) / 1000));
}
