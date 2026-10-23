import { request } from "./client";
import type { Interest, Me, Preferences, Profile, ProfileInput } from "./types";

export const getMe = () => request<Me>("/me");
export const deleteAccount = (password: string) =>
  request("/me", { method: "DELETE", body: { password } });

export const listInterests = async () =>
  (await request<{ interests: Interest[] }>("/interests")).interests;

export const getProfile = () => request<Profile>("/me/profile");
export const saveProfile = (input: ProfileInput) =>
  request<Profile>("/me/profile", { method: "PUT", body: input });

export const getPreferences = () => request<Preferences>("/me/preferences");
export const savePreferences = (input: Preferences) =>
  request<Preferences>("/me/preferences", { method: "PUT", body: input });
