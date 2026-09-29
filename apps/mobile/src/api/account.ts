import { apiRequest } from "./client";
import { endpoints } from "./endpoints";

export type Me = {
  id: string;
  email: string;
  /** "YYYY-MM-DD" */
  birthDate: string;
  age: number;
  createdAt: string;
  profileComplete: boolean;
};

export async function getMe(): Promise<Me> {
  return apiRequest<Me>(endpoints.account.me);
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  await apiRequest<void>(endpoints.account.password, {
    method: "POST",
    body: JSON.stringify({ currentPassword, newPassword })
  });
}

/** Permanently erases the account and everything attached to it. */
export async function deleteAccount(password: string): Promise<void> {
  await apiRequest<void>(endpoints.account.me, {
    method: "DELETE",
    body: JSON.stringify({ password })
  });
}
