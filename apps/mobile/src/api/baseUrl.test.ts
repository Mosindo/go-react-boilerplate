import { resolveBaseUrl, resolveMediaUrl, toWsUrl } from './baseUrl';

describe('resolveBaseUrl', () => {
  it('prefers EXPO_PUBLIC_API_URL and strips trailing slashes', () => {
    expect(resolveBaseUrl({ envUrl: 'http://192.168.1.10:18080/', os: 'ios' })).toBe(
      'http://192.168.1.10:18080',
    );
  });

  it('falls back to localhost:18080, or 10.0.2.2:18080 on the Android emulator', () => {
    expect(resolveBaseUrl({ envUrl: undefined, os: 'ios' })).toBe('http://localhost:18080');
    expect(resolveBaseUrl({ envUrl: '  ', os: 'web' })).toBe('http://localhost:18080');
    expect(resolveBaseUrl({ envUrl: undefined, os: 'android' })).toBe('http://10.0.2.2:18080');
  });
});

describe('resolveMediaUrl', () => {
  it('prefixes relative signed URLs with the API base URL', () => {
    expect(resolveMediaUrl('http://api:1', '/photos/p1/content?exp=1&sig=ab')).toBe(
      'http://api:1/photos/p1/content?exp=1&sig=ab',
    );
    expect(resolveMediaUrl('http://api:1', 'photos/p1/content')).toBe(
      'http://api:1/photos/p1/content',
    );
    expect(resolveMediaUrl('http://api:1', 'https://cdn.example/x.jpg')).toBe(
      'https://cdn.example/x.jpg',
    );
  });
});

describe('toWsUrl', () => {
  it('switches the scheme and encodes the ticket', () => {
    expect(toWsUrl('http://api:1', 'a b')).toBe('ws://api:1/ws?ticket=a%20b');
    expect(toWsUrl('https://api.example.com', 't')).toBe('wss://api.example.com/ws?ticket=t');
  });
});
