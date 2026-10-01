import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AuthSession, AuthUser } from "./types";

export type { AuthSession, AuthUser } from "./types";

export function register(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.register, { method: "POST", body: { email, password }, auth: false });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.login, { method: "POST", body: { email, password }, auth: false });
}

export function logout(refreshToken: string): Promise<void> {
  return apiRequest(endpoints.auth.logout, { method: "POST", body: { refreshToken }, auth: false });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.auth.me);
}

export function requestPasswordReset(email: string): Promise<void> {
  return apiRequest(endpoints.auth.forgot, { method: "POST", body: { email }, auth: false });
}

export function resetPassword(code: string, newPassword: string): Promise<void> {
  return apiRequest(endpoints.auth.reset, { method: "POST", body: { code, newPassword }, auth: false });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return apiRequest(endpoints.auth.changePassword, { method: "POST", body: { currentPassword, newPassword } });
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest(endpoints.account.delete, { method: "DELETE", body: { password } });
}
