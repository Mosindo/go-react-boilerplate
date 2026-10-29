import { request } from "./client";
import type { AuthSession, Me } from "./types";

export const register = (email: string, password: string) =>
  request<AuthSession>("/auth/register", {
    method: "POST",
    body: { email, password },
    auth: false
  });

export const login = (email: string, password: string) =>
  request<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false });

export const logout = (refreshToken: string) =>
  request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false });

export const fetchMe = () => request<Me>("/me");

export const forgotPassword = (email: string) =>
  request<void>("/auth/forgot-password", { method: "POST", body: { email }, auth: false });

export const resetPassword = (email: string, code: string, newPassword: string) =>
  request<void>("/auth/reset-password", {
    method: "POST",
    body: { email, code, newPassword },
    auth: false
  });

export const changePassword = (currentPassword: string, newPassword: string) =>
  request<void>("/me/password", { method: "PUT", body: { currentPassword, newPassword } });

export const deleteAccount = (password: string) =>
  request<void>("/me", { method: "DELETE", body: { password } });
