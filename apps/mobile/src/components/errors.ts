import { ApiError } from "../api/client";

/** HTTP status of an API failure, or null for network/timeouts/unknown errors. */
export function errorStatus(error: unknown): number | null {
  return error instanceof ApiError ? error.status : null;
}

/** Server-provided message (`{error}`), falling back to a generic one. */
export function errorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError && error.status < 500 && error.message) return error.message;
  return fallback;
}
