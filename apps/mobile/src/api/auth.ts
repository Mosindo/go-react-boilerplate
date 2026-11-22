import { api } from "./client";
import type { AuthSession, AuthUser } from "./types";

export function register(email: string, password: string): Promise<AuthSession> {
  return api<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false });
}

export function logout(refreshToken: string): Promise<void> {
  return api<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false });
}

export function me(): Promise<AuthUser> {
  return api<AuthUser>("/me");
}

export function requestPasswordReset(email: string): Promise<void> {
  return api<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false });
}

export function resetPassword(email: string, code: string, password: string): Promise<void> {
  return api<void>("/auth/password/reset", { method: "POST", body: { email, code, password }, auth: false });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return api<void>("/me/password", { method: "POST", body: { currentPassword, newPassword } });
}

export function deleteAccount(password: string): Promise<void> {
  return api<void>("/me", { method: "DELETE", body: { password } });
}
