export const endpoints = {
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    me: "/me"
  },
  users: {
    list: "/users",
    detail: (userId: string) => `/users/${userId}`
  },
  posts: {
    list: "/posts",
    create: "/posts",
    detail: (postId: string) => `/posts/${postId}`
  },
  chat: {
    chats: "/chats",
    messages: (userId: string) => `/chats/${userId}/messages`
  },
  notifications: {
    list: "/notifications",
    create: "/notifications",
    markRead: (notificationId: string) => `/notifications/${notificationId}/read`
  },
  billing: {
    subscription: "/billing/subscription",
    checkout: "/billing/checkout",
    webhook: "/billing/webhook"
  }
} as const;
