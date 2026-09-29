import type { Me } from "./account";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  /** Alias of accessToken kept by the server for older clients. */
  token?: string;
  user: Me;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: Me;
};

function normalizeSession(response: AuthResponse): AuthSession {
  const accessToken = response.accessToken ?? response.token;
  if (!accessToken || !response.refreshToken) {
    throw new Error("The server response did not include a session.");
  }
  return { accessToken, refreshToken: response.refreshToken, user: response.user };
}

function post<T>(path: string, body: unknown): Promise<T> {
  return apiRequest<T>(path, { method: "POST", body: JSON.stringify(body), authenticated: false });
}

export async function register(email: string, password: string, birthDate: string): Promise<AuthSession> {
  return normalizeSession(await post<AuthResponse>(endpoints.auth.register, { email, password, birthDate }));
}

export async function login(email: string, password: string): Promise<AuthSession> {
  return normalizeSession(await post<AuthResponse>(endpoints.auth.login, { email, password }));
}

export async function refreshSession(refreshToken: string): Promise<AuthSession> {
  return normalizeSession(await post<AuthResponse>(endpoints.auth.refresh, { refreshToken }));
}

export async function logout(refreshToken: string): Promise<void> {
  await post<void>(endpoints.auth.logout, { refreshToken });
}

/** Always resolves (202) so it never reveals whether the account exists. */
export async function requestPasswordReset(email: string): Promise<void> {
  await post<void>(endpoints.auth.forgot, { email });
}

export async function resetPassword(token: string, password: string): Promise<void> {
  await post<void>(endpoints.auth.reset, { token, password });
}
