import { apiRequest } from "./client";
import type { AuthUser } from "./types";

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

function toSession(response: AuthResponse): AuthSession {
  const accessToken = response.accessToken ?? response.token;
  if (!accessToken) {
    throw new Error("La réponse d'authentification est invalide.");
  }
  return { accessToken, refreshToken: response.refreshToken, user: response.user };
}

export async function register(email: string, password: string): Promise<AuthSession> {
  return toSession(await apiRequest<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, anonymous: true }));
}

export async function login(email: string, password: string): Promise<AuthSession> {
  return toSession(await apiRequest<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, anonymous: true }));
}

export async function logout(refreshToken: string): Promise<void> {
  await apiRequest("/auth/logout", { method: "POST", body: { refreshToken }, anonymous: true });
}

export function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>("/me");
}

export function forgotPassword(email: string): Promise<void> {
  return apiRequest("/auth/password/forgot", { method: "POST", body: { email }, anonymous: true });
}

export function resetPassword(token: string, newPassword: string): Promise<void> {
  return apiRequest("/auth/password/reset", { method: "POST", body: { token, newPassword }, anonymous: true });
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest("/me", { method: "DELETE", body: { password } });
}
