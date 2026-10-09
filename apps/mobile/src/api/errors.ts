import { fr } from '../i18n/fr';
import { ApiError } from './http';

/** Maps any thrown value to a French, user-facing message. */
export function errorMessage(err: unknown): string {
  if (!(err instanceof ApiError)) return fr.errors.unknown;
  switch (err.code) {
    case 'network':
      return fr.errors.network;
    case 'timeout':
      return fr.errors.timeout;
    case 'unauthorized':
      return fr.errors.unauthorized;
    case 'forbidden':
      return fr.errors.forbidden;
    case 'not_found':
      return fr.errors.notFound;
    case 'conflict':
      return fr.errors.conflict;
    case 'rate_limited':
      return err.retryAfter ? fr.errors.rateLimitedIn(err.retryAfter) : fr.errors.rateLimited;
    case 'invalid_code':
      return fr.errors.invalidCode;
    case 'photo_limit':
      return fr.errors.photoLimit;
    case 'profile_incomplete':
      return fr.errors.profileIncomplete;
    case 'invalid_request':
      return fr.errors.invalid;
    case 'internal':
      return fr.errors.internal;
    default:
      return err.status >= 500 ? fr.errors.internal : fr.errors.unknown;
  }
}

export function isApiError(err: unknown, code?: string): err is ApiError {
  return err instanceof ApiError && (code === undefined || err.code === code);
}
