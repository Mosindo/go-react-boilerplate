import { request } from "./client";
import type { AuthSession, AuthUser } from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) =>
    request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => request<AuthUser>("/me"),
  forgotPassword: (email: string) =>
    request<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    request<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } })
};
