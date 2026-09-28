import { api } from "./http";
import { endpoints } from "./endpoints";
import type { AuthSession, Me } from "./types";

export type { AuthSession } from "./types";

export function register(email: string, password: string): Promise<AuthSession> {
  return api.request<AuthSession>(endpoints.auth.register, { method: "POST", body: { email, password }, auth: false });
}

export function login(email: string, password: string): Promise<AuthSession> {
  return api.request<AuthSession>(endpoints.auth.login, { method: "POST", body: { email, password }, auth: false });
}

export function logout(refreshToken: string): Promise<void> {
  return api.request(endpoints.auth.logout, { method: "POST", body: { refreshToken }, auth: false });
}

export function forgotPassword(email: string): Promise<void> {
  return api.request(endpoints.auth.forgotPassword, { method: "POST", body: { email }, auth: false });
}

export function resetPassword(email: string, code: string, newPassword: string): Promise<void> {
  return api.request(endpoints.auth.resetPassword, {
    method: "POST",
    body: { email, code, newPassword },
    auth: false
  });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return api.request(endpoints.changePassword, { method: "POST", body: { currentPassword, newPassword } });
}

export function deleteAccount(password: string): Promise<void> {
  return api.request(endpoints.me, { method: "DELETE", body: { password } });
}

export function getMe(): Promise<Me> {
  return api.request<Me>(endpoints.me);
}
