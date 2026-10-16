import { endpoints } from "./endpoints";
import { apiRequest } from "./client";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthResponse = {
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

export function normalizeSession(response: AuthResponse): AuthSession {
  const accessToken = response.accessToken ?? response.token;
  if (!accessToken) {
    throw new Error("La réponse d'authentification est incomplète.");
  }
  return { accessToken, refreshToken: response.refreshToken, user: response.user };
}

export async function register(email: string, password: string): Promise<AuthSession> {
  const response = await apiRequest<AuthResponse>(endpoints.auth.register, {
    method: "POST",
    body: JSON.stringify({ email, password }),
    skipAuthRefresh: true
  });
  return normalizeSession(response);
}

export async function login(email: string, password: string): Promise<AuthSession> {
  const response = await apiRequest<AuthResponse>(endpoints.auth.login, {
    method: "POST",
    body: JSON.stringify({ email, password }),
    skipAuthRefresh: true
  });
  return normalizeSession(response);
}

export async function logout(refreshToken: string): Promise<void> {
  await apiRequest(endpoints.auth.logout, {
    method: "POST",
    body: JSON.stringify({ refreshToken }),
    skipAuthRefresh: true
  });
}

export async function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.auth.me, { method: "GET" });
}

export async function requestPasswordReset(email: string): Promise<void> {
  await apiRequest(endpoints.auth.forgotPassword, {
    method: "POST",
    body: JSON.stringify({ email }),
    skipAuthRefresh: true
  });
}

export async function resetPassword(email: string, code: string, newPassword: string): Promise<void> {
  await apiRequest(endpoints.auth.resetPassword, {
    method: "POST",
    body: JSON.stringify({ email, code, newPassword }),
    skipAuthRefresh: true
  });
}

export async function deleteAccount(password: string): Promise<void> {
  await apiRequest(endpoints.auth.me, {
    method: "DELETE",
    body: JSON.stringify({ password }),
    skipAuthRefresh: true
  });
}
