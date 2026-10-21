import { apiRequest } from "./client";
import type { AuthResponse, Me } from "./types";

export const register = (email: string, password: string, birthDate: string) =>
  apiRequest<AuthResponse>("/auth/register", { method: "POST", auth: false, body: { email, password, birthDate } });

export const login = (email: string, password: string) =>
  apiRequest<AuthResponse>("/auth/login", { method: "POST", auth: false, body: { email, password } });

export const logout = (refreshToken: string) => apiRequest("/auth/logout", { method: "POST", auth: false, body: { refreshToken } });

export const forgotPassword = (email: string) => apiRequest("/auth/forgot-password", { method: "POST", auth: false, body: { email } });

export const resetPassword = (code: string, password: string) =>
  apiRequest("/auth/reset-password", { method: "POST", auth: false, body: { code, password } });

export const fetchMe = () => apiRequest<Me>("/me");

export const changePassword = (currentPassword: string, newPassword: string) =>
  apiRequest("/me/password", { method: "POST", body: { currentPassword, newPassword } });

export const deleteAccount = (password: string) => apiRequest("/me", { method: "DELETE", body: { password } });
