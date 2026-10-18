import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AuthUser } from "./types";

export type { AuthUser } from "./types";

type AuthResponse = {
  token?: string;
  accessToken?: string;
  refreshToken: string;
  user: AuthUser;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

function normalizeSession(response: AuthResponse): AuthSession {
  const accessToken = response.accessToken ?? response.token;
  if (!accessToken) {
    throw new Error("La réponse d'authentification est incomplète.");
  }
  return { accessToken, refreshToken: response.refreshToken, user: response.user };
}

export async function register(email: string, password: string): Promise<AuthSession> {
  return normalizeSession(await apiRequest<AuthResponse>(endpoints.auth.register, { method: "POST", body: { email, password } }));
}

export async function login(email: string, password: string): Promise<AuthSession> {
  return normalizeSession(await apiRequest<AuthResponse>(endpoints.auth.login, { method: "POST", body: { email, password } }));
}

export async function logout(refreshToken: string): Promise<void> {
  await apiRequest(endpoints.auth.logout, { method: "POST", body: { refreshToken } });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.auth.me);
}

export async function requestPasswordReset(email: string): Promise<void> {
  await apiRequest(endpoints.auth.forgot, { method: "POST", body: { email } });
}

export async function resetPassword(email: string, code: string, newPassword: string): Promise<void> {
  await apiRequest(endpoints.auth.reset, { method: "POST", body: { email, code, newPassword } });
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  await apiRequest(endpoints.auth.changePassword, { method: "POST", body: { currentPassword, newPassword } });
}

export async function deleteAccount(password: string): Promise<void> {
  await apiRequest(endpoints.auth.me, { method: "DELETE", body: { password } });
}
