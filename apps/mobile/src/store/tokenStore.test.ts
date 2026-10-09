import { createTokenStore } from './tokenStore';

function backend(initial: string | null = null) {
  let value = initial;
  return {
    read: jest.fn(async () => value),
    write: jest.fn(async (v: string) => {
      value = v;
    }),
    remove: jest.fn(async () => {
      value = null;
    }),
  };
}

describe('token store', () => {
  it('loads persisted tokens once, notifies subscribers on set/clear', async () => {
    const b = backend(JSON.stringify({ accessToken: 'a', refreshToken: 'r' }));
    const store = createTokenStore(b);
    expect(store.peek()).toBeNull();
    expect(await store.get()).toEqual({ accessToken: 'a', refreshToken: 'r' });
    await store.get();
    expect(b.read).toHaveBeenCalledTimes(1);
    const listener = jest.fn();
    const off = store.subscribe(listener);
    await store.set({ accessToken: 'b', refreshToken: 's' });
    expect(store.peek()?.accessToken).toBe('b');
    await store.clear();
    expect(store.peek()).toBeNull();
    expect(listener).toHaveBeenCalledTimes(2);
    off();
  });

  it('treats corrupted storage as signed out and survives write failures', async () => {
    const store = createTokenStore({
      read: async () => '{nope',
      write: async () => {
        throw new Error('disk');
      },
      remove: async () => undefined,
    });
    expect(await store.get()).toBeNull();
    await store.set({ accessToken: 'x', refreshToken: 'y' });
    expect(store.peek()?.refreshToken).toBe('y');
  });
});
