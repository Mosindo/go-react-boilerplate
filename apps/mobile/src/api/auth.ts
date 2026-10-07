import { apiRequest } from "./client";
import type { AuthSession, User } from "./types";

export type { AuthSession, User } from "./types";

export function register(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>("/auth/register", {
    method: "POST",
    body: { email, password },
    auth: false
  });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false });
}

export function logout(refreshToken: string): Promise<void> {
  return apiRequest<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false });
}

export function requestPasswordReset(email: string): Promise<{ ok: true }> {
  return apiRequest<{ ok: true }>("/auth/password-reset/request", {
    method: "POST",
    body: { email },
    auth: false
  });
}

export function confirmPasswordReset(token: string, newPassword: string): Promise<void> {
  return apiRequest<void>("/auth/password-reset/confirm", {
    method: "POST",
    body: { token, newPassword },
    auth: false
  });
}

export function me(): Promise<User> {
  return apiRequest<User>("/me");
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest<void>("/me", { method: "DELETE", body: { password } });
}
