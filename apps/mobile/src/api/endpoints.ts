const enc = encodeURIComponent;

export const endpoints = {
  health: "/health",
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    forgotPassword: "/auth/password/forgot",
    resetPassword: "/auth/password/reset",
    me: "/me",
    changePassword: "/me/password"
  },
  profile: {
    me: "/me/profile",
    preferences: "/me/preferences",
    location: "/me/location",
    privacy: "/me/privacy",
    interests: "/interests",
    of: (userId: string) => `/profiles/${enc(userId)}`
  },
  photos: {
    list: "/photos",
    order: "/photos/order",
    one: (id: string) => `/photos/${enc(id)}`,
    main: (id: string) => `/photos/${enc(id)}/main`
  },
  matching: {
    discover: "/discover",
    swipes: "/swipes",
    unmatch: (userId: string) => `/matches/${enc(userId)}`
  },
  chats: {
    list: "/chats",
    one: (id: string) => `/chats/${enc(id)}`,
    messages: (id: string) => `/chats/${enc(id)}/messages`,
    read: (id: string) => `/chats/${enc(id)}/read`
  },
  notifications: {
    list: "/notifications",
    readAll: "/notifications/read-all",
    read: (id: string) => `/notifications/${enc(id)}/read`
  },
  safety: {
    blocks: "/blocks",
    block: (userId: string) => `/blocks/${enc(userId)}`,
    reports: "/reports"
  },
  realtime: {
    ticket: "/ws/ticket",
    socket: "/ws"
  }
} as const;
