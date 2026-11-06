import { fr } from "../i18n/fr";
import { ApiError } from "./client";

/** Maps any thrown value to a friendly French message. */
export function errorMessage(error: unknown, fallback: string = fr.common.genericError): string {
  if (error instanceof ApiError) {
    if (error.isNetwork) return fr.common.networkError;
    const known = fr.errors[error.code];
    if (known) return known;
    if (error.status === 400 || error.status === 409) return error.message;
  }
  return fallback;
}
