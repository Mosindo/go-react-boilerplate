import { api } from "./client";
import type { AuthResponse, AuthUser } from "./types";

export function register(email: string, password: string, acceptedTerms: boolean) {
  return api<AuthResponse>("/auth/register", { method: "POST", body: { email, password, acceptedTerms }, anonymous: true });
}

export function login(email: string, password: string) {
  return api<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, anonymous: true });
}

export function logout(refreshToken: string) {
  return api<void>("/auth/logout", { method: "POST", body: { refreshToken }, anonymous: true });
}

export function me() {
  return api<AuthUser>("/me");
}

export function requestPasswordReset(email: string) {
  return api<void>("/auth/password/forgot", { method: "POST", body: { email }, anonymous: true });
}

export function resetPassword(email: string, code: string, password: string) {
  return api<void>("/auth/password/reset", { method: "POST", body: { email, code, password }, anonymous: true });
}

export function changePassword(currentPassword: string, newPassword: string) {
  return api<void>("/me/password", { method: "PUT", body: { currentPassword, newPassword } });
}

export function deleteAccount(password: string) {
  return api<void>("/me", { method: "DELETE", body: { password } });
}
