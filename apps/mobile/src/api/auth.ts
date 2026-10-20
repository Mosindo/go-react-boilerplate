import { apiRequest } from "./client";
import type { AuthSession, User } from "./types";

export const register = (email: string, password: string) =>
  apiRequest<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false });

export const login = (email: string, password: string) =>
  apiRequest<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false });

export const logout = (refreshToken: string) =>
  apiRequest("/auth/logout", { method: "POST", body: { refreshToken }, auth: false });

export const requestPasswordReset = (email: string) =>
  apiRequest("/auth/password-reset/request", { method: "POST", body: { email }, auth: false });

export const confirmPasswordReset = (token: string, password: string) =>
  apiRequest("/auth/password-reset/confirm", { method: "POST", body: { token, password }, auth: false });

export const fetchMe = () => apiRequest<User>("/me");

export const deleteAccount = (password: string) =>
  apiRequest("/me", { method: "DELETE", body: { password } });
