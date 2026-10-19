import { request } from "./client";
import type { AuthSession, AuthUser } from "./types";

export const register = (email: string, password: string) =>
  request<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false });

export const login = (email: string, password: string) =>
  request<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false });

export const logout = (refreshToken: string) =>
  request("/auth/logout", { method: "POST", body: { refreshToken }, auth: false });

export const me = () => request<AuthUser>("/me");

export const forgotPassword = (email: string) =>
  request("/auth/password/forgot", { method: "POST", body: { email }, auth: false });

export const resetPassword = (email: string, code: string, newPassword: string) =>
  request("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false });

export const deleteAccount = (password: string) =>
  request("/me/delete", { method: "POST", body: { password } });
