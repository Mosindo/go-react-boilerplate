import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AuthUser } from "./types";

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export function register(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.register, { method: "POST", json: { email, password }, anonymous: true });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.login, { method: "POST", json: { email, password }, anonymous: true });
}

export function logout(refreshToken: string): Promise<void> {
  return apiRequest(endpoints.auth.logout, { method: "POST", json: { refreshToken }, anonymous: true });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.auth.me);
}

export function requestPasswordReset(email: string): Promise<void> {
  return apiRequest(endpoints.auth.forgot, { method: "POST", json: { email }, anonymous: true });
}

export function resetPassword(email: string, code: string, newPassword: string): Promise<void> {
  return apiRequest(endpoints.auth.reset, { method: "POST", json: { email, code, newPassword }, anonymous: true });
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest(endpoints.auth.me, { method: "DELETE", json: { password } });
}
