import { apiRequest } from "./client";
import type { AuthSession, AuthUser } from "./types";

export const authApi = {
  register: (email: string, password: string) => apiRequest<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) => apiRequest<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  refresh: (refreshToken: string) => apiRequest<AuthSession>("/auth/refresh", { method: "POST", body: { refreshToken }, auth: false }),
  logout: (refreshToken: string) => apiRequest<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => apiRequest<AuthUser>("/me"),
  forgotPassword: (email: string) => apiRequest<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    apiRequest<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  changePassword: (currentPassword: string, newPassword: string) =>
    apiRequest<void>("/me/password", { method: "POST", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => apiRequest<void>("/me", { method: "DELETE", body: { password } })
};
