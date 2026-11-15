import { api } from "./client";
import type { AuthSession, AuthUser } from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthSession>("/auth/register", { method: "POST", body: { email, password, acceptTerms: true }, auth: false }),
  login: (email: string, password: string) => api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => api<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => api<AuthUser>("/me"),
  forgotPassword: (email: string) => api<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    api<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  changePassword: (currentPassword: string, newPassword: string) =>
    api<void>("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api<void>("/me", { method: "DELETE", body: { password } })
};
