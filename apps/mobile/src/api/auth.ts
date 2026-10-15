import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AuthUser } from "./types";

export type { AuthUser };

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

function toSession(response: AuthResponse): AuthSession {
  return { accessToken: response.accessToken, refreshToken: response.refreshToken, user: response.user };
}

export async function register(email: string, password: string): Promise<AuthSession> {
  const response = await apiRequest<AuthResponse>(endpoints.auth.register, {
    method: "POST",
    body: { email, password },
    auth: false
  });
  return toSession(response);
}

export async function login(email: string, password: string): Promise<AuthSession> {
  const response = await apiRequest<AuthResponse>(endpoints.auth.login, {
    method: "POST",
    body: { email, password },
    auth: false
  });
  return toSession(response);
}

export async function logout(refreshToken: string): Promise<void> {
  await apiRequest(endpoints.auth.logout, { method: "POST", body: { refreshToken }, auth: false });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.auth.me);
}

export function requestPasswordReset(email: string): Promise<void> {
  return apiRequest(endpoints.auth.forgotPassword, { method: "POST", body: { email }, auth: false });
}

export function resetPassword(email: string, code: string, newPassword: string): Promise<void> {
  return apiRequest(endpoints.auth.resetPassword, {
    method: "POST",
    body: { email, code, newPassword },
    auth: false
  });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return apiRequest(endpoints.auth.changePassword, { method: "PUT", body: { currentPassword, newPassword } });
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest(endpoints.auth.me, { method: "DELETE", body: { password } });
}
