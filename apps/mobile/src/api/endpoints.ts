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
    mine: "/profile",
    preferences: "/profile/preferences",
    settings: "/profile/settings",
    location: "/profile/location",
    photos: "/profile/photos",
    photosOrder: "/profile/photos/order",
    photo: (photoId: string) => `/profile/photos/${photoId}`,
    public: (userId: string) => `/profiles/${userId}`,
    interests: "/interests"
  },
  discovery: {
    deck: (limit: number) => `/discover?limit=${limit}`,
    swipe: "/swipes"
  },
  matches: {
    list: (limit: number, offset: number) => `/matches?limit=${limit}&offset=${offset}`,
    remove: (matchId: string) => `/matches/${matchId}`
  },
  chat: {
    messages: (conversationId: string, before?: string) =>
      `/conversations/${conversationId}/messages?limit=30${before ? `&before=${before}` : ""}`,
    send: (conversationId: string) => `/conversations/${conversationId}/messages`,
    read: (conversationId: string) => `/conversations/${conversationId}/read`
  },
  safety: {
    blocks: "/blocks",
    unblock: (userId: string) => `/blocks/${userId}`,
    reports: "/reports"
  },
  notifications: {
    list: (offset: number) => `/notifications?limit=20&offset=${offset}`,
    unreadCount: "/notifications/unread-count",
    readAll: "/notifications/read-all",
    read: (id: string) => `/notifications/${id}/read`
  }
} as const;
