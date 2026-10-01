export const endpoints = {
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    forgot: "/auth/password/forgot",
    reset: "/auth/password/reset",
    changePassword: "/me/password",
    me: "/me"
  },
  profile: {
    own: "/me/profile",
    preferences: "/me/preferences",
    location: "/me/location",
    interests: "/interests",
    public: (userId: string) => `/profiles/${userId}`
  },
  photos: {
    list: "/me/photos",
    order: "/me/photos/order",
    item: (photoId: string) => `/me/photos/${photoId}`
  },
  discover: {
    list: "/discover",
    swipe: "/swipes",
    matches: "/matches",
    unmatch: (matchId: string) => `/matches/${matchId}`
  },
  chat: {
    conversations: "/conversations",
    messages: (conversationId: string) => `/conversations/${conversationId}/messages`,
    read: (conversationId: string) => `/conversations/${conversationId}/read`,
    clear: (conversationId: string) => `/conversations/${conversationId}`
  },
  notifications: {
    list: "/notifications",
    markRead: (notificationId: string) => `/notifications/${notificationId}/read`,
    readAll: "/notifications/read-all"
  },
  safety: {
    blocks: "/blocks",
    unblock: (userId: string) => `/blocks/${userId}`,
    reports: "/reports"
  },
  account: { delete: "/me" },
  realtime: { ticket: "/ws/ticket", socket: "/ws" }
} as const;
