/** Every backend path in one place (see docs/API.md). */
export const endpoints = {
  auth: {
    register: "/auth/register",
    login: "/auth/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    forgot: "/auth/forgot",
    reset: "/auth/reset"
  },
  account: {
    me: "/me",
    password: "/me/password"
  },
  profile: {
    interests: "/interests",
    mine: "/me/profile",
    location: "/me/location",
    preferences: "/me/preferences",
    detail: (userId: string) => `/profiles/${encodeURIComponent(userId)}`
  },
  photos: {
    upload: "/me/photos",
    order: "/me/photos/order",
    remove: (photoId: string) => `/me/photos/${encodeURIComponent(photoId)}`
  },
  safety: {
    blocks: "/blocks",
    unblock: (userId: string) => `/blocks/${encodeURIComponent(userId)}`,
    reports: "/reports"
  }
} as const;
