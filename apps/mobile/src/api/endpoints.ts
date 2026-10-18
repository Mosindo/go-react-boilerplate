export const endpoints = {
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    forgot: "/auth/password/forgot",
    reset: "/auth/password/reset",
    me: "/me",
    changePassword: "/me/password"
  },
  profile: {
    me: "/me/profile",
    settings: "/me/settings",
    interests: "/interests",
    photos: "/me/photos",
    photosOrder: "/me/photos/order",
    photo: (photoId: string) => `/me/photos/${photoId}`,
    other: (userId: string) => `/profiles/${userId}`
  },
  discovery: {
    discover: "/discover",
    swipes: "/swipes",
    resetPasses: "/swipes/passes",
    unmatch: (matchId: string) => `/matches/${matchId}`
  },
  chat: {
    conversations: "/conversations",
    conversation: (id: string) => `/conversations/${id}`,
    messages: (id: string) => `/conversations/${id}/messages`,
    read: (id: string) => `/conversations/${id}/read`
  },
  notifications: {
    list: "/notifications",
    readAll: "/notifications/read-all",
    read: (id: string) => `/notifications/${id}/read`
  },
  safety: {
    blocks: "/blocks",
    unblock: (userId: string) => `/blocks/${userId}`,
    reports: "/reports"
  },
  realtime: {
    ticket: "/realtime/ticket",
    socket: "/realtime/ws"
  }
} as const;
