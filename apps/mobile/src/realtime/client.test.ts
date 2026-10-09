import { RealtimeClient, backoffDelay, parseFrame, type SocketLike } from './client';

class FakeSocket implements SocketLike {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: unknown }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  closed = false;
  constructor(public url: string) {}
  close() {
    this.closed = true;
  }
}

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve();
};

describe('parseFrame', () => {
  it('accepts known frames and rejects garbage', () => {
    expect(parseFrame('{"type":"match.new","data":{"matchId":"1"}}')).toEqual({
      type: 'match.new',
      data: { matchId: '1' },
    });
    expect(parseFrame('{"type":"unknown","data":{}}')).toBeNull();
    expect(parseFrame('not json')).toBeNull();
    expect(parseFrame('{"type":"message.new"}')).toBeNull();
    expect(parseFrame(42)).toBeNull();
  });
});

describe('backoffDelay', () => {
  it('grows exponentially with jitter and is capped', () => {
    expect(backoffDelay(0, 1000, 30000, () => 1)).toBe(1000);
    expect(backoffDelay(3, 1000, 30000, () => 1)).toBe(8000);
    expect(backoffDelay(3, 1000, 30000, () => 0)).toBe(4000);
    expect(backoffDelay(10, 1000, 30000, () => 1)).toBe(30000);
  });
});

describe('RealtimeClient', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  function setup() {
    const sockets: FakeSocket[] = [];
    let ticketN = 0;
    const getTicket = jest.fn(async () => `t${++ticketN}`);
    const onEvent = jest.fn();
    const onOpen = jest.fn();
    const client = new RealtimeClient({
      getTicket,
      wsBaseUrl: 'ws://api.test',
      onEvent,
      onOpen,
      createSocket: (url) => {
        const s = new FakeSocket(url);
        sockets.push(s);
        return s;
      },
      random: () => 1,
    });
    return { client, sockets, getTicket, onEvent, onOpen };
  }

  it('connects with a fresh ticket, dispatches events and reports open', async () => {
    const { client, sockets, onEvent, onOpen } = setup();
    client.start();
    await flush();
    expect(sockets[0]?.url).toBe('ws://api.test/ws?ticket=t1');
    sockets[0]!.onopen?.({});
    expect(onOpen).toHaveBeenCalledWith(false);
    sockets[0]!.onmessage?.({ data: '{"type":"message.new","data":{"id":"1"}}' });
    expect(onEvent).toHaveBeenCalledWith({ type: 'message.new', data: { id: '1' } });
    client.stop();
  });

  it('reconnects with backoff and a NEW ticket after a close, flagged as reconnect', async () => {
    const { client, sockets, getTicket, onOpen } = setup();
    client.start();
    await flush();
    sockets[0]!.onopen?.({});
    sockets[0]!.onclose?.({});
    expect(sockets[0]!.closed).toBe(true);
    await jest.advanceTimersByTimeAsync(999);
    expect(getTicket).toHaveBeenCalledTimes(1);
    await jest.advanceTimersByTimeAsync(2);
    await flush();
    expect(getTicket).toHaveBeenCalledTimes(2);
    expect(sockets[1]?.url).toBe('ws://api.test/ws?ticket=t2');
    sockets[1]!.onopen?.({});
    expect(onOpen).toHaveBeenLastCalledWith(true);
    client.stop();
  });

  it('backs off again when the ticket request fails, and stop() cancels everything', async () => {
    const { client, getTicket, sockets } = setup();
    getTicket.mockRejectedValueOnce(new Error('offline'));
    client.start();
    await flush();
    expect(sockets).toHaveLength(0);
    await jest.advanceTimersByTimeAsync(1100);
    await flush();
    expect(sockets).toHaveLength(1);
    client.stop();
    sockets[0]!.onclose?.({}); // ignored after stop
    await jest.advanceTimersByTimeAsync(60_000);
    expect(getTicket).toHaveBeenCalledTimes(2);
  });

  it('reconnectNow() skips the wait when disconnected and is a no-op when open', async () => {
    const { client, sockets, getTicket } = setup();
    client.start();
    await flush();
    sockets[0]!.onopen?.({});
    client.reconnectNow();
    expect(getTicket).toHaveBeenCalledTimes(1);
    sockets[0]!.onclose?.({});
    client.reconnectNow();
    await flush();
    expect(getTicket).toHaveBeenCalledTimes(2);
    client.stop();
  });
});
