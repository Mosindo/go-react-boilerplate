export const endpoints = {
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    me: "/me",
    requestReset: "/auth/password-reset/request",
    confirmReset: "/auth/password-reset/confirm"
  },
  profile: {
    own: "/me/profile",
    location: "/me/location",
    preferences: "/me/preferences",
    privacy: "/me/privacy",
    interests: "/interests",
    public: (userId: string) => `/profiles/${userId}`
  },
  photos: {
    list: "/me/photos",
    order: "/me/photos/order",
    item: (photoId: string) => `/me/photos/${photoId}`
  },
  discovery: {
    discover: "/discover",
    swipe: "/swipes"
  },
  conversations: {
    list: "/conversations",
    messages: (matchId: string) => `/conversations/${matchId}/messages`,
    read: (matchId: string) => `/conversations/${matchId}/read`,
    hide: (matchId: string) => `/conversations/${matchId}`,
    unmatch: (matchId: string) => `/matches/${matchId}`
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
  }
} as const;
