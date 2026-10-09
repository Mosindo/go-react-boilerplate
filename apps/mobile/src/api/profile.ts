import { Platform } from 'react-native';
import { api } from './http';
import type { Interest, MyProfile, Photo, Preferences, ProfileInput } from './types';

export function profileToInput(p: MyProfile, patch: Partial<ProfileInput> = {}): ProfileInput {
  return {
    firstName: p.firstName,
    birthDate: p.birthDate,
    gender: p.gender,
    bio: p.bio,
    city: p.city,
    latitude: p.latitude,
    longitude: p.longitude,
    interests: p.interests.map((i) => i.slug),
    discoverable: p.discoverable,
    showDistance: p.showDistance,
    ...patch,
  };
}

/** Builds the multipart body. Native uses the RN `{uri,name,type}` file descriptor, web a Blob. */
export async function buildPhotoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === 'web') {
    const blob = await (await fetch(uri)).blob();
    form.append('file', blob, 'photo.jpg');
  } else {
    form.append('file', { uri, name: 'photo.jpg', type: 'image/jpeg' } as unknown as Blob);
  }
  return form;
}

export const profileApi = {
  interests: () => api.get<{ interests: Interest[] }>('/interests').then((r) => r.interests),
  getProfile: (signal?: AbortSignal) => api.get<MyProfile>('/me/profile', { signal }),
  saveProfile: (input: ProfileInput) => api.put<MyProfile>('/me/profile', input),
  getPreferences: () => api.get<Preferences>('/me/preferences'),
  savePreferences: (p: Preferences) => api.put<Preferences>('/me/preferences', p),
  listPhotos: () => api.get<{ photos: Photo[] }>('/me/photos').then((r) => r.photos),
  uploadPhoto: async (uri: string) => api.upload<Photo>('/me/photos', await buildPhotoForm(uri)),
  reorderPhotos: (ids: string[]) =>
    api.put<{ photos: Photo[] }>('/me/photos/order', { ids }).then((r) => r.photos),
  setMainPhoto: (id: string) =>
    api.post<{ photos: Photo[] }>(`/me/photos/${id}/main`).then((r) => r.photos),
  deletePhoto: (id: string) => api.delete<void>(`/me/photos/${id}`),
};
