import { http } from "./client";
import type { AuthResponse, Me } from "./types";

export type Credentials = { email: string; password: string };

export const authApi = {
  register: ({ email, password }: Credentials) =>
    http.post<AuthResponse>("/auth/register", { email, password }, { auth: false }),
  login: ({ email, password }: Credentials) =>
    http.post<AuthResponse>("/auth/login", { email, password }, { auth: false }),
  logout: (refreshToken: string) => http.post("/auth/logout", { refreshToken }, { auth: false }),
  forgotPassword: (email: string) => http.post("/auth/password/forgot", { email }, { auth: false }),
  resetPassword: (input: { email: string; code: string; password: string }) =>
    http.post("/auth/password/reset", input, { auth: false }),
  me: (signal?: AbortSignal) => http.get<Me>("/me", undefined, signal),
  changePassword: (input: { currentPassword: string; newPassword: string }) => http.post("/me/password", input),
  deleteAccount: (password: string) => http.delete("/me", { password })
};
