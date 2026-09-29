import { API_BASE_URL, apiRequest } from "./client";
import { buildWsUrlFromBase } from "../lib/dating/ws";

export async function fetchWsTicket(): Promise<string> {
  const res = await apiRequest<{ ticket: string; expiresInSeconds: number }>("/ws/ticket", {
    method: "POST"
  });
  if (!res?.ticket) throw new Error("Realtime ticket missing");
  return res.ticket;
}

/** http(s)://host → ws(s)://host/ws?ticket=… */
export function buildWsUrl(ticket: string): string {
  return buildWsUrlFromBase(API_BASE_URL, ticket);
}
