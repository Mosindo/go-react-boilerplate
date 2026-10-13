export const endpoints = {
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    forgot: "/auth/password/forgot",
    reset: "/auth/password/reset",
    me: "/me"
  },
  profile: {
    own: "/profile",
    preferences: "/profile/preferences",
    location: "/profile/location",
    interests: "/interests",
    public: (userId: string) => `/profiles/${userId}`
  },
  photos: {
    list: "/photos",
    order: "/photos/order",
    item: (photoId: string) => `/photos/${photoId}`,
    file: (photoId: string, size: "thumb" | "full") => `/photos/${photoId}/file${size === "thumb" ? "?size=thumb" : ""}`
  },
  discover: "/discover",
  likes: "/likes",
  matches: {
    list: "/matches",
    item: (matchId: string) => `/matches/${matchId}`
  },
  conversations: {
    list: "/conversations",
    item: (id: string) => `/conversations/${id}`,
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
  socket: "/ws"
} as const;
