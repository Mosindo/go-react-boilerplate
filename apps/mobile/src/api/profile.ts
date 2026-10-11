import { http } from './client';
import type {
  Interest,
  LocationInput,
  OwnProfile,
  Preferences,
  Privacy,
  ProfileInput,
  PublicProfile,
} from './types';

export const profileApi = {
  async interests(): Promise<Interest[]> {
    const res = await http.request<{ interests: Interest[] }>('GET', '/interests');
    return res.interests;
  },

  /** Rejects with a 404 `profile_not_found` ApiError when onboarding hasn't saved a profile yet. */
  own(): Promise<OwnProfile> {
    return http.request<OwnProfile>('GET', '/me/profile');
  },

  save(input: ProfileInput): Promise<OwnProfile | undefined> {
    return http.request<OwnProfile | undefined>('PUT', '/me/profile', { body: input });
  },

  setLocation(input: LocationInput): Promise<void> {
    return http.request<void>('PUT', '/me/location', { body: input });
  },

  clearLocation(): Promise<void> {
    return http.request<void>('DELETE', '/me/location');
  },

  preferences(): Promise<Preferences> {
    return http.request<Preferences>('GET', '/me/preferences');
  },

  savePreferences(input: Preferences): Promise<Preferences | undefined> {
    return http.request<Preferences | undefined>('PUT', '/me/preferences', { body: input });
  },

  privacy(): Promise<Privacy> {
    return http.request<Privacy>('GET', '/me/privacy');
  },

  savePrivacy(input: Privacy): Promise<Privacy | undefined> {
    return http.request<Privacy | undefined>('PUT', '/me/privacy', { body: input });
  },

  publicProfile(userId: string): Promise<PublicProfile> {
    return http.request<PublicProfile>('GET', `/profiles/${encodeURIComponent(userId)}`);
  },
};
