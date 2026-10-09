import { api } from './http';
import type { AuthResponse, Me } from './types';

export const authApi = {
  register: (email: string, password: string) =>
    api.post<AuthResponse>('/auth/register', { email, password }, { auth: false }),
  login: (email: string, password: string) =>
    api.post<AuthResponse>('/auth/login', { email, password }, { auth: false }),
  logout: (refreshToken: string) =>
    api.post<void>('/auth/logout', { refreshToken }, { auth: false, timeoutMs: 4000 }),
  forgotPassword: (email: string) =>
    api.post<unknown>('/auth/password/forgot', { email }, { auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    api.post<void>('/auth/password/reset', { email, code, newPassword }, { auth: false }),
  me: (signal?: AbortSignal) => api.get<Me>('/me', { signal }),
  changePassword: (currentPassword: string, newPassword: string) =>
    api.post<void>('/me/password', { currentPassword, newPassword }),
  deleteAccount: (password: string) => api.delete<void>('/me', { password }),
};
