import { api } from "./client";
import type { Conversation, Message, MessagesPage, Page } from "./types";

export const chatApi = {
  conversations: (cursor?: string | null) =>
    api<Page<"conversations", Conversation>>(`/conversations${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`),
  conversation: (id: string) => api<Conversation>(`/conversations/${id}`),
  messages: (id: string, cursor?: string | null) =>
    api<MessagesPage>(`/conversations/${id}/messages?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
  send: (id: string, body: string) => api<Message>(`/conversations/${id}/messages`, { method: "POST", body: { body } }),
  markRead: (id: string) => api<{ readAt: string }>(`/conversations/${id}/read`, { method: "POST" }),
  clear: (id: string) => api<void>(`/conversations/${id}`, { method: "DELETE" })
};
