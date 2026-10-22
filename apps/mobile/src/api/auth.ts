import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AuthResponse, User } from "./types";

export function register(email: string, password: string): Promise<AuthResponse> {
  return apiRequest<AuthResponse>(endpoints.register, { method: "POST", body: { email, password }, auth: false, quiet: true });
}

export function login(email: string, password: string): Promise<AuthResponse> {
  return apiRequest<AuthResponse>(endpoints.login, { method: "POST", body: { email, password }, auth: false, quiet: true });
}

export function logout(refreshToken: string): Promise<void> {
  return apiRequest(endpoints.logout, { method: "POST", body: { refreshToken }, auth: false, quiet: true });
}

export function forgotPassword(email: string): Promise<void> {
  return apiRequest(endpoints.forgotPassword, { method: "POST", body: { email }, auth: false, quiet: true });
}

export function resetPassword(token: string, password: string): Promise<void> {
  return apiRequest(endpoints.resetPassword, { method: "POST", body: { token, password }, auth: false, quiet: true });
}

export function fetchMe(): Promise<User> {
  return apiRequest<User>(endpoints.me);
}

export function deleteAccount(password: string): Promise<void> {
  return apiRequest(endpoints.me, { method: "DELETE", body: { password }, quiet: true });
}
