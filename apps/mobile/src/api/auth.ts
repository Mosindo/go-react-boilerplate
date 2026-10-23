import { request } from "./client";
import type { AuthResponse } from "./types";

export const register = (email: string, password: string) =>
  request<AuthResponse>("/auth/register", {
    method: "POST",
    body: { email, password },
    auth: false
  });

export const login = (email: string, password: string) =>
  request<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false });

export const logout = (refreshToken: string) =>
  request("/auth/logout", { method: "POST", body: { refreshToken }, auth: false });

export const forgotPassword = (email: string) =>
  request("/auth/forgot-password", { method: "POST", body: { email }, auth: false });

export const resetPassword = (email: string, code: string, password: string) =>
  request("/auth/reset-password", {
    method: "POST",
    body: { email, code, password },
    auth: false
  });
