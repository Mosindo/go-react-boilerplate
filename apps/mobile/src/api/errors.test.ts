import { fr } from '../i18n';
import { ApiError, errorFromResponse, errorMessage, isNotFound, networkError } from './errors';

describe('errorFromResponse', () => {
  it('reads {error, code} bodies', () => {
    const err = errorFromResponse(409, { error: 'email already used', code: 'email_taken' });
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(409);
    expect(err.code).toBe('email_taken');
  });

  it('derives a code when the body is not the documented shape', () => {
    expect(errorFromResponse(401, undefined).code).toBe('unauthorized');
    expect(errorFromResponse(404, '<html>').code).toBe('not_found');
    expect(errorFromResponse(502, null).code).toBe('unknown');
  });
});

describe('errorMessage', () => {
  it('maps API codes to French copy, never exposing server text', () => {
    const err = new ApiError(401, 'invalid_credentials', 'bad creds');
    expect(errorMessage(err)).toBe(fr.errors.invalid_credentials);
    expect(errorMessage(err)).not.toContain('bad creds');
    expect(errorMessage(new ApiError(400, 'underage', 'x'))).toBe(fr.errors.underage);
  });

  it('is friendly on network failures, server errors and unknown throwables', () => {
    expect(errorMessage(networkError())).toBe(fr.errors.network);
    expect(errorMessage(networkError(true))).toBe(fr.errors.timeout);
    expect(errorMessage(new ApiError(503, 'weird', 'x'))).toBe(fr.errors.server);
    expect(errorMessage(new Error('boom'))).toBe(fr.errors.unknown);
    expect(errorMessage('string')).toBe(fr.errors.unknown);
  });

  it('detects 404s', () => {
    expect(isNotFound(new ApiError(404, 'profile_not_found', 'x'))).toBe(true);
    expect(isNotFound(new ApiError(500, 'x', 'x'))).toBe(false);
    expect(isNotFound(null)).toBe(false);
  });
});
