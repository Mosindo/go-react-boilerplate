import { PING_INTERVAL_MS, RealtimeClient } from './client';
import type { SocketHandlers, SocketHandle } from './client';

class FakeSocket implements SocketHandle {
  sent: string[] = [];
  closed = false;
  constructor(
    public url: string,
    public handlers: SocketHandlers,
  ) {}
  send(data: string) {
    this.sent.push(data);
  }
  close() {
    this.closed = true;
  }
}

function setup(ticketFails: () => boolean = () => false) {
  const sockets: FakeSocket[] = [];
  const events: string[] = [];
  let tickets = 0;
  const reconnected = jest.fn();
  const client = new RealtimeClient({
    getUrl: async () => {
      if (ticketFails()) throw new Error('ticket failed');
      tickets += 1;
      return `ws://x/ws?ticket=t${tickets}`;
    },
    openSocket: (url, handlers) => {
      const s = new FakeSocket(url, handlers);
      sockets.push(s);
      return s;
    },
    onEvent: (e) => events.push(e.type),
    onReconnected: reconnected,
    random: () => 0.5,
  });
  return { client, sockets, events, reconnected };
}

const flush = async () => {
  for (let i = 0; i < 5; i += 1) await Promise.resolve();
};

describe('RealtimeClient', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('opens with a fresh ticket, forwards events and pings every 25s', async () => {
    const { client, sockets, events } = setup();
    client.start();
    await flush();
    expect(sockets[0].url).toBe('ws://x/ws?ticket=t1');
    sockets[0].handlers.onOpen();
    expect(client.getStatus()).toBe('open');
    sockets[0].handlers.onMessage(
      JSON.stringify({ type: 'conversation.removed', data: { conversationId: 'c' } }),
    );
    sockets[0].handlers.onMessage(JSON.stringify({ type: 'pong', data: {} }));
    expect(events).toEqual(['conversation.removed']);
    jest.advanceTimersByTime(PING_INTERVAL_MS * 2);
    expect(sockets[0].sent).toEqual(['{"type":"ping"}', '{"type":"ping"}']);
    client.stop();
  });

  it('reconnects with exponential backoff, a new ticket, and signals the refetch', async () => {
    const { client, sockets, reconnected } = setup();
    client.start();
    await flush();
    sockets[0].handlers.onOpen();
    expect(reconnected).not.toHaveBeenCalled();

    sockets[0].handlers.onClose();
    expect(client.getStatus()).toBe('reconnecting');
    jest.advanceTimersByTime(999);
    await flush();
    expect(sockets).toHaveLength(1);
    jest.advanceTimersByTime(1);
    await flush();
    expect(sockets).toHaveLength(2);
    expect(sockets[1].url).toContain('ticket=t2');

    // Second drop before opening: delay doubles.
    sockets[1].handlers.onClose();
    jest.advanceTimersByTime(1999);
    await flush();
    expect(sockets).toHaveLength(2);
    jest.advanceTimersByTime(1);
    await flush();
    expect(sockets).toHaveLength(3);

    sockets[2].handlers.onOpen();
    expect(reconnected).toHaveBeenCalledTimes(1);
    client.stop();
  });

  it('retries when the ticket request fails', async () => {
    let fail = true;
    const { client, sockets } = setup(() => fail);
    client.start();
    await flush();
    expect(sockets).toHaveLength(0);
    fail = false;
    jest.advanceTimersByTime(1000);
    await flush();
    expect(sockets).toHaveLength(1);
    client.stop();
  });

  it('stop() closes the socket, cancels retries and ignores late events', async () => {
    const { client, sockets, events } = setup();
    client.start();
    await flush();
    sockets[0].handlers.onOpen();
    client.stop();
    expect(sockets[0].closed).toBe(true);
    sockets[0].handlers.onMessage(
      JSON.stringify({ type: 'conversation.removed', data: { conversationId: 'c' } }),
    );
    sockets[0].handlers.onClose();
    jest.advanceTimersByTime(60_000);
    await flush();
    expect(sockets).toHaveLength(1);
    expect(events).toEqual([]);
    expect(client.getStatus()).toBe('idle');
  });

  it('ensureConnected() reconnects immediately when the socket is down', async () => {
    const { client, sockets } = setup();
    client.start();
    await flush();
    sockets[0].handlers.onOpen();
    sockets[0].handlers.onClose();
    client.ensureConnected();
    await flush();
    expect(sockets).toHaveLength(2);
    client.stop();
  });
});
