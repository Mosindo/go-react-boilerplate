import { type AuthUser } from "./auth";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";

export type PlatformUser = AuthUser;

export type Post = {
  id: string;
  authorUserId: string;
  title: string;
  body: string;
  createdAt: string;
  updatedAt: string;
};

export type ChatMessage = {
  id: string;
  senderUserId: string;
  recipientUserId: string;
  content: string;
  createdAt: string;
};

export type ChatSummary = {
  user: PlatformUser;
  lastMessage?: {
    content: string;
    createdAt: string;
  };
};

export type Notification = {
  id: string;
  userId: string;
  type: string;
  title: string;
  body: string;
  isRead: boolean;
  createdAt: string;
  readAt?: string;
};

export type BillingSubscription = {
  id?: string;
  organizationId: string;
  provider: string;
  status: string;
  stripeCustomerId?: string;
  stripeSubscriptionId?: string;
  stripeCheckoutSessionId?: string;
  currentPeriodStart?: string;
  currentPeriodEnd?: string;
  cancelAtPeriodEnd: boolean;
  canceledAt?: string;
  createdAt?: string;
  updatedAt?: string;
};

export type BillingCheckoutSession = {
  sessionId: string;
  checkoutUrl: string;
  organizationId: string;
  status: string;
};

type UsersResponse = {
  users: PlatformUser[];
};

type PostsResponse = {
  posts: Post[];
};

type ChatsResponse = {
  chats: ChatSummary[];
};

type ChatMessagesResponse = {
  messages: ChatMessage[];
};

type NotificationsResponse = {
  notifications: Notification[];
};

export type CreateNotificationInput = {
  type: string;
  title: string;
  body: string;
};

function authHeaders(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

export async function listUsers(token: string): Promise<PlatformUser[]> {
  const payload = await apiRequest<UsersResponse>(endpoints.users.list, {
    method: "GET",
    headers: authHeaders(token)
  });
  return payload.users;
}

export async function listPosts(token: string): Promise<Post[]> {
  const payload = await apiRequest<PostsResponse>(endpoints.posts.list, {
    method: "GET",
    headers: authHeaders(token)
  });
  return payload.posts;
}

export async function createPost(token: string, title: string, body: string): Promise<Post> {
  return apiRequest<Post>(endpoints.posts.create, {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify({ title, body })
  });
}

export async function listChats(token: string): Promise<ChatSummary[]> {
  const payload = await apiRequest<ChatsResponse>(endpoints.chat.chats, {
    method: "GET",
    headers: authHeaders(token)
  });
  return payload.chats;
}

export async function listChatMessages(token: string, userId: string): Promise<ChatMessage[]> {
  const payload = await apiRequest<ChatMessagesResponse>(endpoints.chat.messages(userId), {
    method: "GET",
    headers: authHeaders(token)
  });
  return payload.messages;
}

export async function sendChatMessage(token: string, userId: string, content: string): Promise<ChatMessage> {
  return apiRequest<ChatMessage>(endpoints.chat.messages(userId), {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify({ content })
  });
}

export async function listNotifications(token: string): Promise<Notification[]> {
  const payload = await apiRequest<NotificationsResponse>(endpoints.notifications.list, {
    method: "GET",
    headers: authHeaders(token)
  });
  return payload.notifications;
}

export async function createNotification(token: string, input: CreateNotificationInput): Promise<Notification> {
  return apiRequest<Notification>(endpoints.notifications.create, {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify(input)
  });
}

export async function markNotificationRead(token: string, notificationId: string): Promise<Notification> {
  return apiRequest<Notification>(endpoints.notifications.markRead(notificationId), {
    method: "POST",
    headers: authHeaders(token)
  });
}

export async function getBillingSubscription(token: string): Promise<BillingSubscription> {
  return apiRequest<BillingSubscription>(endpoints.billing.subscription, {
    method: "GET",
    headers: authHeaders(token)
  });
}

export async function createBillingCheckout(token: string): Promise<BillingCheckoutSession> {
  return apiRequest<BillingCheckoutSession>(endpoints.billing.checkout, {
    method: "POST",
    headers: authHeaders(token)
  });
}
