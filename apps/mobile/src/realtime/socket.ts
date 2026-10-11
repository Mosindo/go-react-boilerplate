import type { SocketHandle, SocketHandlers } from './client';

/** Real WebSocket adapter (React Native and web share the global WebSocket). */
export function openWebSocket(url: string, handlers: SocketHandlers): SocketHandle {
  const ws = new WebSocket(url);
  ws.onopen = () => handlers.onOpen();
  ws.onmessage = (event) => handlers.onMessage(event.data);
  ws.onclose = () => handlers.onClose();
  ws.onerror = () => {
    // The close event always follows an error and drives the reconnect.
  };
  return {
    send: (data) => ws.send(data),
    close: () => {
      ws.onclose = null;
      ws.close();
    },
  };
}
