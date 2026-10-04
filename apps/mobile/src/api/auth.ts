import { api } from "./client";
import type { AuthResponse, Me } from "./types";

async function storeSession(data: AuthResponse): Promise<AuthResponse> {
  await api.tokens.set({ accessToken: data.accessToken, refreshToken: data.refreshToken });
  return data;
}

export async function register(email: string, password: string): Promise<AuthResponse> {
  return storeSession(await api.post<AuthResponse>("/auth/register", { email, password }, { auth: false }));
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  return storeSession(await api.post<AuthResponse>("/auth/login", { email, password }, { auth: false }));
}

/** Best effort: always clears the local session even when the server call fails. */
export async function logout(): Promise<void> {
  const refreshToken = api.tokens.getTokens()?.refreshToken;
  try {
    if (refreshToken) await api.post<void>("/auth/logout", { refreshToken }, { auth: false });
  } catch {
    /* the token will expire on its own */
  } finally {
    await api.tokens.clear();
  }
}

export function forgotPassword(email: string): Promise<void> {
  return api.post<void>("/auth/forgot-password", { email }, { auth: false });
}

export function resetPassword(token: string, password: string): Promise<void> {
  return api.post<void>("/auth/reset-password", { token, password }, { auth: false });
}

export function getMe(): Promise<Me> {
  return api.get<Me>("/me");
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return api.post<void>("/me/password", { currentPassword, newPassword });
}

export async function deleteAccount(password: string): Promise<void> {
  await api.delete<void>("/me", { password });
  await api.tokens.clear();
}
