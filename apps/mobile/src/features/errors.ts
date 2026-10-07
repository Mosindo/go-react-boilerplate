import { ApiError } from "../api/client";

export function apiErrorCode(error: unknown): string | null {
  if (error instanceof ApiError && typeof error.data === "object" && error.data !== null) {
    const code = (error.data as { code?: unknown }).code;
    return typeof code === "string" ? code : null;
  }
  return null;
}

export function isApiErrorCode(error: unknown, code: string): boolean {
  return apiErrorCode(error) === code;
}

export function errorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
