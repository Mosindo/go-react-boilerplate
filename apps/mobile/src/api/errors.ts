import { fr } from '../i18n';
import type { ApiErrorBody, ApiErrorCode, ClientErrorCode } from './types';

export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode | ClientErrorCode | string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }

  get isNetwork(): boolean {
    return this.code === 'network' || this.code === 'timeout';
  }
}

export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError;
}

export function isNotFound(err: unknown): boolean {
  return isApiError(err) && err.status === 404;
}

export function networkError(timeout = false): ApiError {
  return new ApiError(0, timeout ? 'timeout' : 'network', fr.errors.network);
}

function isErrorBody(value: unknown): value is ApiErrorBody {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as { code?: unknown }).code === 'string' &&
    typeof (value as { error?: unknown }).error === 'string'
  );
}

/** Builds an ApiError from a failed HTTP response body (already parsed, may be anything). */
export function errorFromResponse(status: number, body: unknown): ApiError {
  if (isErrorBody(body)) {
    return new ApiError(status, body.code, body.error);
  }
  const code = status === 401 ? 'unauthorized' : status === 404 ? 'not_found' : 'unknown';
  return new ApiError(status, code, `HTTP ${status}`);
}

const BY_CODE: Record<string, string> = {
  invalid_request: fr.errors.invalid_request,
  unauthorized: fr.errors.unauthorized,
  forbidden: fr.errors.forbidden,
  not_found: fr.errors.not_found,
  conflict: fr.errors.conflict,
  rate_limited: fr.errors.rate_limited,
  profile_required: fr.errors.profile_required,
  profile_not_found: fr.errors.profile_not_found,
  invalid_credentials: fr.errors.invalid_credentials,
  email_taken: fr.errors.email_taken,
  underage: fr.errors.underage,
  photo_limit: fr.errors.photo_limit,
  invalid_image: fr.errors.invalid_image,
  blocked: fr.errors.blocked,
  network: fr.errors.network,
  timeout: fr.errors.timeout,
};

/** Maps any thrown value to a friendly French message. Never exposes raw server text. */
export function errorMessage(err: unknown): string {
  if (isApiError(err)) {
    const mapped = BY_CODE[err.code];
    if (mapped) return mapped;
    if (err.status >= 500) return fr.errors.server;
  }
  return fr.errors.unknown;
}
