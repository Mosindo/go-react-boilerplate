import { ApiError } from "../api/errors";
import { t } from "./fr";

/** Maps an error (ApiError code first, then status) to French copy. Backend text is never shown. */
export function errorMessage(error: unknown): string {
  if (!(error instanceof ApiError)) return t.errors.generic;
  switch (error.code) {
    case "network_error":
      return t.errors.network;
    case "timeout":
      return t.errors.timeout;
    case "email_taken":
      return t.errors.emailTaken;
    case "photo_limit":
      return t.errors.photoLimit;
    case "profile_not_found":
      return t.errors.profileNotFound;
    case "invalid_code":
    case "invalid_reset_code":
      return t.errors.invalidCode;
    case "account_suspended":
    case "suspended":
      return t.errors.suspended;
    case "invalid_password":
    case "wrong_password":
      return t.errors.wrongPassword;
    case "validation_failed":
    case "invalid_request":
      return t.errors.invalidRequest;
    default:
      break;
  }
  switch (error.status) {
    case 0:
      return t.errors.network;
    case 400:
      return t.errors.invalidRequest;
    case 401:
      return t.errors.unauthorized;
    case 403:
      return t.errors.forbidden;
    case 404:
      return t.errors.notFound;
    case 413:
      return t.errors.tooLarge;
    case 415:
      return t.errors.unsupportedMedia;
    case 429:
      return error.retryAfterSeconds
        ? t.errors.rateLimitedIn(Math.ceil(error.retryAfterSeconds))
        : t.errors.rateLimited;
    default:
      return error.status >= 500 ? t.errors.server : t.errors.generic;
  }
}
