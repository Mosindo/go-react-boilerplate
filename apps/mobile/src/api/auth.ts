import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AuthResponse } from "./types";

export type AuthSession = AuthResponse;

export function register(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.register, { method: "POST", body: { email, password }, auth: false });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.login, { method: "POST", body: { email, password }, auth: false });
}

export function logout(refreshToken: string): Promise<void> {
  return apiRequest(endpoints.auth.logout, { method: "POST", body: { refreshToken }, auth: false });
}

export function forgotPassword(email: string): Promise<void> {
  return apiRequest(endpoints.auth.forgot, { method: "POST", body: { email }, auth: false });
}

export function resetPassword(email: string, code: string, newPassword: string): Promise<void> {
  return apiRequest(endpoints.auth.reset, { method: "POST", body: { email, code, newPassword }, auth: false });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return apiRequest(endpoints.auth.changePassword, { method: "POST", body: { currentPassword, newPassword } });
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest(endpoints.me, { method: "DELETE", body: { password } });
}
