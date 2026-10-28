import { apiRequest } from "./client";
import type { Account, AuthResponse } from "./types";

export const register = (email: string, password: string) =>
  apiRequest<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false });

export const login = (email: string, password: string) =>
  apiRequest<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false });

export const logout = (refreshToken: string) =>
  apiRequest("/auth/logout", { method: "POST", body: { refreshToken }, auth: false });

export const forgotPassword = (email: string) =>
  apiRequest("/auth/forgot-password", { method: "POST", body: { email }, auth: false });

export const resetPassword = (email: string, code: string, password: string) =>
  apiRequest("/auth/reset-password", { method: "POST", body: { email, code, password }, auth: false });

export const fetchMe = () => apiRequest<Account>("/me");

export const changePassword = (currentPassword: string, newPassword: string) =>
  apiRequest("/me/password", { method: "POST", body: { currentPassword, newPassword } });

export const deleteAccount = (password: string) =>
  apiRequest("/me", { method: "DELETE", body: { password } });
