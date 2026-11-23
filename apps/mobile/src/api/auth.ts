import { apiRequest } from "./client";
import { endpoints } from "./endpoints";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export function register(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.register, { method: "POST", body: { email, password }, auth: false });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.login, { method: "POST", body: { email, password }, auth: false });
}

export function refreshSession(refreshToken: string): Promise<AuthSession> {
  return apiRequest<AuthSession>(endpoints.auth.refresh, { method: "POST", body: { refreshToken }, auth: false });
}

export function logout(refreshToken: string): Promise<void> {
  return apiRequest<void>(endpoints.auth.logout, { method: "POST", body: { refreshToken }, auth: false });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.auth.me);
}

export function requestPasswordReset(email: string): Promise<void> {
  return apiRequest<void>(endpoints.auth.forgot, { method: "POST", body: { email }, auth: false });
}

export function resetPassword(email: string, code: string, password: string): Promise<void> {
  return apiRequest<void>(endpoints.auth.reset, { method: "POST", body: { email, code, password }, auth: false });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return apiRequest<void>(endpoints.auth.changePassword, { method: "POST", body: { currentPassword, newPassword } });
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest<void>(endpoints.auth.me, { method: "DELETE", body: { password } });
}
