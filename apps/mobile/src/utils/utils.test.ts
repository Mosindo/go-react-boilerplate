import { computeResize } from './image';
import { formatDayHeader, formatListTime, isSameDay, localId } from './format';
import { resolveApiBaseUrl, toWebSocketUrl } from '../config';
import { errorMessage } from '../api/errors';
import { ApiError } from '../api/http';

describe('computeResize', () => {
  it('leaves small images alone and caps the long side at 1600', () => {
    expect(computeResize(1200, 900)).toBeNull();
    expect(computeResize(4000, 3000)).toEqual({ width: 1600 });
    expect(computeResize(3000, 4000)).toEqual({ height: 1600 });
    expect(computeResize(0, 0)).toBeNull();
  });
});

describe('format helpers', () => {
  const now = new Date(2026, 9, 9, 15, 0);
  it('formats list timestamps relative to today', () => {
    expect(formatListTime(new Date(2026, 9, 9, 8, 5).toISOString(), now)).toMatch(/08:05|8:05/);
    expect(formatListTime(new Date(2026, 9, 8, 22, 0).toISOString(), now)).toBe('Hier');
    expect(formatListTime(new Date(2026, 8, 1, 22, 0).toISOString(), now)).toMatch(/1/);
  });
  it('formats day headers and compares days', () => {
    expect(formatDayHeader(new Date(2026, 9, 9, 1).toISOString(), now)).toBe('Aujourd’hui');
    expect(formatDayHeader(new Date(2026, 9, 8, 1).toISOString(), now)).toBe('Hier');
    expect(
      isSameDay(new Date(2026, 9, 9, 1).toISOString(), new Date(2026, 9, 9, 23).toISOString()),
    ).toBe(true);
    expect(
      isSameDay(new Date(2026, 9, 9, 1).toISOString(), new Date(2026, 9, 10, 1).toISOString()),
    ).toBe(false);
  });
  it('generates unique local ids', () => {
    expect(localId()).not.toBe(localId());
  });
});

describe('config', () => {
  it('resolves the API base URL', () => {
    expect(resolveApiBaseUrl('http://192.168.1.5:18080/', 'ios', true)).toBe(
      'http://192.168.1.5:18080',
    );
    expect(resolveApiBaseUrl(undefined, 'android', true)).toBe('http://10.0.2.2:18080');
    expect(resolveApiBaseUrl('', 'ios', true)).toBe('http://localhost:18080');
    expect(resolveApiBaseUrl(undefined, 'ios', false)).toBe('');
  });
  it('derives ws urls', () => {
    expect(toWebSocketUrl('http://localhost:18080')).toBe('ws://localhost:18080');
    expect(toWebSocketUrl('https://api.lumo.app')).toBe('wss://api.lumo.app');
  });
});

describe('errorMessage', () => {
  it('maps API codes to French messages', () => {
    expect(errorMessage(new ApiError(0, 'network', 'x'))).toMatch(/serveur/);
    expect(errorMessage(new ApiError(429, 'rate_limited', 'x', 7))).toMatch(/7 s/);
    expect(errorMessage(new ApiError(400, 'invalid_code', 'x'))).toMatch(/code/);
    expect(errorMessage(new ApiError(409, 'photo_limit', 'x'))).toMatch(/6 photos/);
    expect(errorMessage(new ApiError(502, 'weird', 'x'))).toMatch(/erreur/);
    expect(errorMessage(new Error('boom'))).toMatch(/inattendue/);
  });
});
