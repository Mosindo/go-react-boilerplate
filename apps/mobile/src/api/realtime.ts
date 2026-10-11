import { http } from './client';
import type { WsTicket } from './types';

export const realtimeApi = {
  ticket(): Promise<WsTicket> {
    return http.request<WsTicket>('POST', '/ws/ticket');
  },
};
