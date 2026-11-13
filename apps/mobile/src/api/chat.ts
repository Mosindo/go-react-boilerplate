import { request } from "./client";
import type { Conversation, Message, MessagesPage } from "./types";

export const chatApi = {
  conversations: () => request<{ chats: Conversation[] }>("/chats?limit=100").then((r) => r.chats),
  messages: (userId: string, before?: string) =>
    request<MessagesPage>(`/chats/${userId}/messages?limit=30${before ? `&before=${before}` : ""}`),
  send: (userId: string, content: string) =>
    request<Message>(`/chats/${userId}/messages`, { method: "POST", body: { content } }),
  markRead: (userId: string) => request<{ readAt: string }>(`/chats/${userId}/read`, { method: "POST" }),
  clear: (userId: string) => request<void>(`/chats/${userId}`, { method: "DELETE" })
};
