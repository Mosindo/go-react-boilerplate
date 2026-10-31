import { apiRequest } from "./client";
import type { AuthSession, AuthUser } from "./types";

export function register(email: string, password: string, birthDate: string): Promise<AuthSession> {
  return apiRequest<AuthSession>("/auth/register", { method: "POST", body: { email, password, birthDate }, noAuth: true, silent: true });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>("/auth/login", { method: "POST", body: { email, password }, noAuth: true, silent: true });
}

export function logout(refreshToken: string): Promise<void> {
  return apiRequest("/auth/logout", { method: "POST", body: { refreshToken }, noAuth: true, silent: true });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>("/me", { silent: true });
}

export function forgotPassword(email: string): Promise<void> {
  return apiRequest("/auth/password/forgot", { method: "POST", body: { email }, noAuth: true, silent: true });
}

export function resetPassword(email: string, code: string, newPassword: string): Promise<void> {
  return apiRequest("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, noAuth: true, silent: true });
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest("/me", { method: "DELETE", body: { password }, silent: true });
}
