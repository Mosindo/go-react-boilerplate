import { api } from "./client";
import { endpoints } from "./endpoints";
import type { AuthResponse, AuthUser } from "./types";
import { tokenStore } from "../store/tokenStore";

export async function register(email: string, password: string): Promise<AuthResponse> {
  const session = await api.request<AuthResponse>(endpoints.register, { method: "POST", body: { email, password }, auth: false });
  await tokenStore.save({ accessToken: session.accessToken, refreshToken: session.refreshToken });
  return session;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const session = await api.request<AuthResponse>(endpoints.login, { method: "POST", body: { email, password }, auth: false });
  await tokenStore.save({ accessToken: session.accessToken, refreshToken: session.refreshToken });
  return session;
}

export function fetchMe(signal?: AbortSignal): Promise<AuthUser> {
  return api.request<AuthUser>(endpoints.me, { signal });
}

/** Revokes the server session, then forgets the tokens locally even if the network call fails. */
export async function logout(): Promise<void> {
  const tokens = await tokenStore.load();
  try {
    if (tokens) {
      await api.request<void>(endpoints.logout, { method: "POST", body: { refreshToken: tokens.refreshToken }, auth: false });
    }
  } catch {
    // Offline logout still signs out locally; the server session expires on its own.
  } finally {
    await tokenStore.clear();
  }
}

export function forgotPassword(email: string): Promise<void> {
  return api.request<void>(endpoints.forgotPassword, { method: "POST", body: { email }, auth: false });
}

export function resetPassword(token: string, password: string): Promise<void> {
  return api.request<void>(endpoints.resetPassword, { method: "POST", body: { token, password }, auth: false });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return api.request<void>(endpoints.changePassword, { method: "POST", body: { currentPassword, newPassword } });
}

export async function deleteAccount(password: string): Promise<void> {
  await api.request<void>(endpoints.me, { method: "DELETE", body: { password } });
  await tokenStore.clear();
}
