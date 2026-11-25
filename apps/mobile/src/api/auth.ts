import { api } from "./client";
import type { AuthResponse, AuthUser } from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) => api<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => api<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) => api<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (code: string, password: string) =>
    api<void>("/auth/password/reset", { method: "POST", body: { code, password }, auth: false }),
  me: () => api<AuthUser>("/me"),
  changePassword: (currentPassword: string, newPassword: string) =>
    api<void>("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api<void>("/me", { method: "DELETE", body: { password } })
};
