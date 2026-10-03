import { apiRequest } from "./client";
import { endpoints } from "./endpoints";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type AuthSession = AuthResponse;

const jsonPost = (body: unknown) => ({ method: "POST", body: JSON.stringify(body) });

export function register(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthResponse>(endpoints.auth.register, { ...jsonPost({ email, password }), silent: true });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return apiRequest<AuthResponse>(endpoints.auth.login, { ...jsonPost({ email, password }), silent: true });
}

export function refreshSession(refreshToken: string): Promise<AuthSession> {
  return apiRequest<AuthResponse>(endpoints.auth.refresh, { ...jsonPost({ refreshToken }), skipRefresh: true, silent: true });
}

export async function logout(refreshToken: string): Promise<void> {
  await apiRequest(endpoints.auth.logout, { ...jsonPost({ refreshToken }), silent: true });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.auth.me, { method: "GET", silent: true });
}

export async function requestPasswordReset(email: string): Promise<void> {
  await apiRequest(endpoints.auth.requestReset, { ...jsonPost({ email }), silent: true });
}

export async function confirmPasswordReset(token: string, newPassword: string): Promise<void> {
  await apiRequest(endpoints.auth.confirmReset, { ...jsonPost({ token, newPassword }), silent: true });
}

export async function deleteAccount(password: string): Promise<void> {
  await apiRequest(endpoints.auth.me, { method: "DELETE", body: JSON.stringify({ password }), silent: true });
}
