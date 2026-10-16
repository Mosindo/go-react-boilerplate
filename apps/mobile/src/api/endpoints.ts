export const endpoints = {
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    forgotPassword: "/auth/forgot-password",
    resetPassword: "/auth/reset-password",
    me: "/me"
  },
  profile: {
    me: "/me/profile",
    preferences: "/me/preferences",
    privacy: "/me/privacy",
    interests: "/interests",
    detail: (userId: string) => `/profiles/${userId}`
  },
  photos: {
    upload: "/me/photos",
    item: (photoId: string) => `/me/photos/${photoId}`,
    main: (photoId: string) => `/me/photos/${photoId}/main`,
    order: "/me/photos/order"
  },
  discover: "/discover",
  swipes: "/swipes",
  matches: {
    list: "/matches",
    item: (matchId: string) => `/matches/${matchId}`,
    likesReceived: "/likes/received"
  },
  chat: {
    conversations: "/conversations",
    unread: "/conversations/unread-count",
    messages: (matchId: string) => `/conversations/${matchId}/messages`,
    read: (matchId: string) => `/conversations/${matchId}/read`,
    item: (matchId: string) => `/conversations/${matchId}`
  },
  safety: {
    blocks: "/blocks",
    unblock: (userId: string) => `/blocks/${userId}`,
    reports: "/reports"
  },
  notifications: {
    list: "/notifications",
    readAll: "/notifications/read-all",
    markRead: (notificationId: string) => `/notifications/${notificationId}/read`
  },
  realtime: {
    ticket: "/realtime/ticket",
    socket: "/realtime"
  }
} as const;
