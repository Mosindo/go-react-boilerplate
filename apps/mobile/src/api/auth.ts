import { http } from './client';
import type {
  AuthResponse,
  ChangePasswordInput,
  Credentials,
  ResetPasswordInput,
  User,
} from './types';

async function startSession(res: AuthResponse): Promise<User> {
  await http.setSession({ accessToken: res.accessToken, refreshToken: res.refreshToken });
  return res.user;
}

export const authApi = {
  async register(credentials: Credentials): Promise<User> {
    const res = await http.request<AuthResponse>('POST', '/auth/register', {
      body: credentials,
      auth: false,
    });
    return startSession(res);
  },

  async login(credentials: Credentials): Promise<User> {
    const res = await http.request<AuthResponse>('POST', '/auth/login', {
      body: credentials,
      auth: false,
    });
    return startSession(res);
  },

  /** Best effort server-side revoke, then always forget the local session. */
  async logout(): Promise<void> {
    const refreshToken = http.getRefreshToken();
    try {
      if (refreshToken) {
        await http.request<void>('POST', '/auth/logout', {
          body: { refreshToken },
          auth: 'optional',
        });
      }
    } catch {
      // Offline or already revoked: the local session is dropped regardless.
    } finally {
      await http.clearSession();
    }
  },

  forgotPassword(email: string): Promise<void> {
    return http.request<void>('POST', '/auth/forgot-password', { body: { email }, auth: false });
  },

  resetPassword(input: ResetPasswordInput): Promise<void> {
    return http.request<void>('POST', '/auth/reset-password', { body: input, auth: false });
  },

  me(): Promise<User> {
    return http.request<User>('GET', '/me');
  },

  changePassword(input: ChangePasswordInput): Promise<void> {
    return http.request<void>('POST', '/me/password', { body: input });
  },

  async deleteAccount(password: string): Promise<void> {
    await http.request<void>('DELETE', '/me', { body: { password } });
    await http.clearSession();
  },
};
