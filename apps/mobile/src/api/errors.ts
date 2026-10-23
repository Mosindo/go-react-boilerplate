export type ApiErrorBody = {
  error?: string;
  code?: string;
  fields?: Record<string, string>;
};

/** Error raised for every non-2xx response and for transport failures (status 0). */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: Record<string, string>;
  readonly retryAfterSeconds: number | null;

  constructor(
    status: number,
    code: string,
    message: string,
    fields: Record<string, string> = {},
    retryAfterSeconds: number | null = null
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.fields = fields;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

export const NETWORK_ERROR_CODE = "network_error";
export const TIMEOUT_ERROR_CODE = "timeout";

function parseRetryAfter(value: string | null): number | null {
  if (!value) return null;
  const seconds = Number(value);
  return Number.isFinite(seconds) && seconds >= 0 ? seconds : null;
}

/** Maps an HTTP error response (status, parsed JSON body or null, Retry-After header) to ApiError. */
export function mapErrorResponse(
  status: number,
  body: unknown,
  retryAfterHeader: string | null = null
): ApiError {
  const parsed = (body && typeof body === "object" ? body : {}) as ApiErrorBody;
  const code = typeof parsed.code === "string" && parsed.code ? parsed.code : defaultCode(status);
  const message =
    typeof parsed.error === "string" && parsed.error ? parsed.error : `HTTP ${status}`;
  const fields: Record<string, string> = {};
  if (parsed.fields && typeof parsed.fields === "object") {
    for (const [key, value] of Object.entries(parsed.fields)) {
      if (typeof value === "string") fields[key] = value;
    }
  }
  return new ApiError(status, code, message, fields, parseRetryAfter(retryAfterHeader));
}

function defaultCode(status: number): string {
  switch (status) {
    case 400:
      return "invalid_request";
    case 401:
      return "unauthorized";
    case 403:
      return "forbidden";
    case 404:
      return "not_found";
    case 409:
      return "conflict";
    case 413:
      return "payload_too_large";
    case 415:
      return "unsupported_media_type";
    case 429:
      return "rate_limited";
    default:
      return status >= 500 ? "server_error" : "unknown";
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
