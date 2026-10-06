import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AuthSession, AuthUser } from "./types";

type AuthResponse = {
  token?: string;
  accessToken?: string;
  refreshToken: string;
  user: AuthUser;
};

function normalizeSession(response: AuthResponse): AuthSession {
  const accessToken = response.accessToken ?? response.token;
  if (!accessToken) {
    throw new Error("authentication response did not include an access token");
  }
  return { accessToken, refreshToken: response.refreshToken, user: response.user };
}

export async function register(email: string, password: string): Promise<AuthSession> {
  return normalizeSession(
    await apiRequest<AuthResponse>(endpoints.register, { method: "POST", body: { email, password }, anonymous: true })
  );
}

export async function login(email: string, password: string): Promise<AuthSession> {
  return normalizeSession(await apiRequest<AuthResponse>(endpoints.login, { method: "POST", body: { email, password }, anonymous: true }));
}

export async function logout(refreshToken: string): Promise<void> {
  await apiRequest(endpoints.logout, { method: "POST", body: { refreshToken }, anonymous: true });
}

export async function me(): Promise<AuthUser> {
  return apiRequest<AuthUser>(endpoints.me);
}

export async function forgotPassword(email: string): Promise<void> {
  await apiRequest(endpoints.forgotPassword, { method: "POST", body: { email }, anonymous: true });
}

export async function resetPassword(input: { email: string; code: string; newPassword: string }): Promise<void> {
  await apiRequest(endpoints.resetPassword, { method: "POST", body: input, anonymous: true });
}
